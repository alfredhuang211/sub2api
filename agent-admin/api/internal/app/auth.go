package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const subjectContextKey contextKey = "agent_admin_subject"

type Subject struct {
	User        User
	Agent       *AgentProfile
	IsBaseAdmin bool
	IsAdmin     bool
}

type JWTClaims struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenVersion int64  `json:"token_version"`
	jwt.RegisteredClaims
}

func AuthMiddleware(cfg Config, repo *Repository, requireAdmin bool, requireAgent bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, err := authenticate(r.Context(), cfg, repo, r.Header.Get("Authorization"))
		if err != nil {
			WriteError(w, http.StatusUnauthorized, 401, err.Error())
			return
		}
		if requireAdmin && !subject.IsAdmin {
			WriteError(w, http.StatusForbidden, 403, "admin permission required")
			return
		}
		if requireAgent {
			if subject.Agent == nil || subject.Agent.Status != "active" {
				WriteError(w, http.StatusForbidden, 403, "active agent permission required")
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), subjectContextKey, subject)))
	})
}

func SubjectFromContext(ctx context.Context) (Subject, bool) {
	subject, ok := ctx.Value(subjectContextKey).(Subject)
	return subject, ok
}

func authenticate(ctx context.Context, cfg Config, repo *Repository, authHeader string) (Subject, error) {
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return Subject{}, errors.New("JWT_SECRET is required")
	}
	parts := strings.SplitN(strings.TrimSpace(authHeader), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return Subject{}, errors.New("Authorization header format must be Bearer token")
	}

	claims, err := parseJWT(cfg, strings.TrimSpace(parts[1]))
	if err != nil {
		return Subject{}, err
	}

	user, err := repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Subject{}, errors.New("user not found")
		}
		return Subject{}, fmt.Errorf("load user: %w", err)
	}
	if user.Status != "active" {
		return Subject{}, errors.New("user is not active")
	}
	if claims.TokenVersion != resolvedTokenVersion(user) {
		return Subject{}, errors.New("token has been revoked")
	}

	agent, err := repo.GetAgentByUserID(ctx, user.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Subject{}, fmt.Errorf("load agent: %w", err)
	}

	isBaseAdmin := user.Role == "admin"
	isAdmin := isBaseAdmin
	if !isAdmin {
		ok, err := repo.IsAgentAdminUser(ctx, user.ID)
		if err != nil {
			return Subject{}, fmt.Errorf("load admin permission: %w", err)
		}
		isAdmin = ok
	}

	return Subject{User: user, Agent: agent, IsBaseAdmin: isBaseAdmin, IsAdmin: isAdmin}, nil
}

func parseJWT(cfg Config, tokenString string) (*JWTClaims, error) {
	if len(tokenString) > 8192 {
		return nil, errors.New("token too large")
	}
	method := strings.ToUpper(strings.TrimSpace(cfg.JWTSigningMethod))
	if method == "" {
		method = "HS256"
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{method}))
	token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func resolvedTokenVersion(user User) int64 {
	material := strings.ToLower(strings.TrimSpace(user.Email)) + "\n" + user.PasswordHash
	sum := sha256.Sum256([]byte(material))
	fingerprint := int64(binary.BigEndian.Uint64(sum[:8]) & 0x7fffffffffffffff)
	return user.TokenVersion ^ fingerprint
}
