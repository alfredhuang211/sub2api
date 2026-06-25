package app

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr             string
	DatabaseURL          string
	JWTSecret            string
	JWTSigningMethod     string
	Sub2APIBaseURL       string
	Sub2APIProxyPrefixes []string
	MigrationEnabled     bool
	SchedulerEnabled     bool
	FreezeDays           int
	MinSettlementAmount  int64
	SettlementTimezone   string
}

func LoadConfig() Config {
	return Config{
		HTTPAddr:             getenv("AGENT_ADMIN_API_ADDR", "127.0.0.1:3101"),
		DatabaseURL:          strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:            strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTSigningMethod:     getenv("JWT_SIGNING_METHOD", "HS256"),
		Sub2APIBaseURL:       getenv("SUB2API_BASE_URL", "http://sub2api:8080"),
		Sub2APIProxyPrefixes: splitCSV(getenv("SUB2API_PROXY_PREFIXES", "/api/v1/auth,/api/auth")),
		MigrationEnabled:     getenvBool("MIGRATION_ENABLED", true),
		SchedulerEnabled:     getenvBool("SCHEDULER_ENABLED", true),
		FreezeDays:           getenvInt("COMMISSION_FREEZE_DAYS", 5),
		MinSettlementAmount:  int64(getenvInt("MIN_SETTLEMENT_AMOUNT", 10000)),
		SettlementTimezone:   getenv("SETTLEMENT_TIMEZONE", "Asia/Shanghai"),
	}
}

func (c Config) Location() *time.Location {
	loc, err := time.LoadLocation(c.SettlementTimezone)
	if err != nil {
		return time.Local
	}
	return loc
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
