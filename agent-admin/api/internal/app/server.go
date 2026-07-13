package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	cfg   Config
	repo  *Repository
	proxy *httputil.ReverseProxy
}

func NewServer(cfg Config, repo *Repository) (*Server, error) {
	proxy, err := newSub2APIProxy(cfg.Sub2APIBaseURL)
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, repo: repo, proxy: proxy}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/api/v1/health", s.health)
	mux.HandleFunc("/api/", s.routeAPI)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]string{"service": "agent-admin-api", "status": "ok"})
}

func (s *Server) routeAPI(w http.ResponseWriter, r *http.Request) {
	if s.shouldProxyToSub2API(r.URL.Path) {
		s.proxy.ServeHTTP(w, r)
		return
	}
	if s.repo == nil {
		WriteError(w, http.StatusServiceUnavailable, 503, "DATABASE_URL is required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	if path == r.URL.Path {
		WriteError(w, http.StatusNotFound, 404, "not found")
		return
	}

	switch {
	case path == "/settings/public" && r.Method == http.MethodGet:
		settings, err := s.repo.GetPublicSettings(r.Context())
		writeResult(w, settings, err)
	case path == "/me" && r.Method == http.MethodGet:
		s.withAuth(false, false, http.HandlerFunc(s.currentUser)).ServeHTTP(w, r)
	case strings.HasPrefix(path, "/admin/"):
		s.withAuth(true, false, http.HandlerFunc(s.routeAdmin)).ServeHTTP(w, r)
	case strings.HasPrefix(path, "/agent/"):
		s.withAuth(false, true, http.HandlerFunc(s.routeAgent)).ServeHTTP(w, r)
	default:
		WriteError(w, http.StatusNotFound, 404, "not found")
	}
}

func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) {
	subject, _ := SubjectFromContext(r.Context())
	WriteOK(w, CurrentUser{
		ID:          subject.User.ID,
		Email:       subject.User.Email,
		Username:    subject.User.Username,
		Role:        subject.User.Role,
		Status:      subject.User.Status,
		IsBaseAdmin: subject.IsBaseAdmin,
		IsAdmin:     subject.IsAdmin,
		IsAgent:     subject.Agent != nil && subject.Agent.Status == "active",
		Agent:       subject.Agent,
	})
}

func (s *Server) withAuth(requireAdmin, requireAgent bool, next http.Handler) http.Handler {
	return AuthMiddleware(s.cfg, s.repo, requireAdmin, requireAgent, next)
}

func (s *Server) routeAdmin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin")
	subject, _ := SubjectFromContext(r.Context())

	if path == "/agents/summary" && r.Method == http.MethodGet {
		summary, err := s.repo.AdminSummary(r.Context())
		writeResult(w, summary, err)
		return
	}
	if path == "/settings" {
		if !subject.IsBaseAdmin {
			WriteError(w, http.StatusForbidden, 403, "base admin permission required")
			return
		}
		switch r.Method {
		case http.MethodGet:
			settings, err := s.repo.GetSettings(r.Context())
			writeResult(w, settings, err)
		case http.MethodPut:
			var req struct {
				TurnstileEnabled bool   `json:"turnstile_enabled"`
				TurnstileSiteKey string `json:"turnstile_site_key"`
			}
			if !decodeJSON(w, r, &req) {
				return
			}
			settings, err := s.repo.UpdateSettings(r.Context(), AgentAdminSettings{
				TurnstileEnabled: req.TurnstileEnabled,
				TurnstileSiteKey: req.TurnstileSiteKey,
			}, subject.User.ID)
			writeResult(w, settings, err)
		default:
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
		}
		return
	}
	if path == "/admin-users" && r.Method == http.MethodGet {
		if !subject.IsBaseAdmin {
			WriteError(w, http.StatusForbidden, 403, "base admin permission required")
			return
		}
		filter := parseListFilter(r)
		items, total, err := s.repo.ListAgentAdminUsers(r.Context(), filter)
		writePage(w, items, total, filter, err)
		return
	}
	if path == "/admin-users" && r.Method == http.MethodPost {
		if !subject.IsBaseAdmin {
			WriteError(w, http.StatusForbidden, 403, "base admin permission required")
			return
		}
		var req struct {
			UserID int64 `json:"user_id"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		item, err := s.repo.GrantAgentAdmin(r.Context(), req.UserID, subject.User.ID)
		writeResult(w, item, err)
		return
	}
	if strings.HasPrefix(path, "/admin-users/") && strings.HasSuffix(path, "/revoke") && r.Method == http.MethodPost {
		if !subject.IsBaseAdmin {
			WriteError(w, http.StatusForbidden, 403, "base admin permission required")
			return
		}
		idText := strings.TrimSuffix(strings.TrimPrefix(path, "/admin-users/"), "/revoke")
		id, ok := parseID(w, idText)
		if !ok {
			return
		}
		item, err := s.repo.RevokeAgentAdmin(r.Context(), id, subject.User.ID)
		writeResult(w, item, err)
		return
	}
	if path == "/admin-users/candidates" && r.Method == http.MethodGet {
		if !subject.IsBaseAdmin {
			WriteError(w, http.StatusForbidden, 403, "base admin permission required")
			return
		}
		filter := parseListFilter(r)
		items, total, err := s.repo.SearchNonAdminUsers(r.Context(), filter)
		writePage(w, items, total, filter, err)
		return
	}
	if path == "/users/assignable" && r.Method == http.MethodGet {
		filter := parseListFilter(r)
		items, total, err := s.repo.SearchAssignableUsers(r.Context(), filter)
		writePage(w, items, total, filter, err)
		return
	}
	if path == "/users/search" && r.Method == http.MethodGet {
		filter := parseListFilter(r)
		items, total, err := s.repo.SearchUsers(r.Context(), filter)
		writePage(w, items, total, filter, err)
		return
	}
	if path == "/agents" {
		switch r.Method {
		case http.MethodGet:
			items, total, err := s.repo.ListAgents(r.Context(), parseListFilter(r))
			writePage(w, items, total, parseListFilter(r), err)
		case http.MethodPost:
			var req struct {
				UserID        int64  `json:"user_id"`
				Level         int    `json:"level"`
				ParentAgentID *int64 `json:"parent_agent_id"`
				RateBPS       *int   `json:"rate_bps"`
			}
			if !decodeJSON(w, r, &req) {
				return
			}
			agent, err := s.repo.CreateAgent(r.Context(), CreateAgentInput{UserID: req.UserID, Level: req.Level, ParentAgentID: req.ParentAgentID, RateBPS: req.RateBPS, OperatorID: subject.User.ID})
			writeResult(w, agent, err)
		default:
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
		}
		return
	}
	if strings.HasPrefix(path, "/agents/") {
		s.routeAdminAgent(w, r, strings.TrimPrefix(path, "/agents/"), subject)
		return
	}
	if path == "/agent-customer-relations" && r.Method == http.MethodPost {
		var req struct {
			CustomerUserID int64  `json:"customer_user_id"`
			AgentID        int64  `json:"agent_id"`
			Reason         string `json:"reason"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		customer, err := s.repo.AssignCustomer(r.Context(), req.CustomerUserID, req.AgentID, subject.User.ID, req.Reason)
		writeResult(w, customer, err)
		return
	}
	if path == "/agent-customer-relations/changes" && r.Method == http.MethodGet {
		filter := parseListFilter(r)
		items, total, err := s.repo.ListCustomerRelationChanges(r.Context(), filter)
		writePage(w, items, total, filter, err)
		return
	}
	if path == "/agent-commissions" && r.Method == http.MethodGet {
		filter := parseListFilter(r)
		items, total, err := s.repo.ListCommissions(r.Context(), filter)
		writePage(w, items, total, filter, err)
		return
	}
	if path == "/agent-settlements" && r.Method == http.MethodGet {
		filter := parseListFilter(r)
		items, total, err := s.repo.ListSettlements(r.Context(), filter)
		writePage(w, items, total, filter, err)
		return
	}
	if strings.HasPrefix(path, "/agent-settlements/") && strings.HasSuffix(path, "/register-payment") && r.Method == http.MethodPost {
		idText := strings.TrimSuffix(strings.TrimPrefix(path, "/agent-settlements/"), "/register-payment")
		id, ok := parseID(w, idText)
		if !ok {
			return
		}
		var req struct {
			Amount           int64      `json:"amount"`
			PaymentMethod    *string    `json:"payment_method"`
			PaymentReference *string    `json:"payment_reference"`
			PaidAt           *time.Time `json:"paid_at"`
			Remark           *string    `json:"remark"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		item, err := s.repo.RegisterSettlementPayment(r.Context(), id, RegisterSettlementPaymentInput{
			Amount: req.Amount, PaymentMethod: req.PaymentMethod, PaymentReference: req.PaymentReference,
			PaidAt: req.PaidAt, Remark: req.Remark, OperatorID: subject.User.ID,
		})
		writeResult(w, item, err)
		return
	}
	if strings.HasPrefix(path, "/agent-settlements/") && strings.HasSuffix(path, "/adjust") && r.Method == http.MethodPost {
		idText := strings.TrimSuffix(strings.TrimPrefix(path, "/agent-settlements/"), "/adjust")
		id, ok := parseID(w, idText)
		if !ok {
			return
		}
		var req struct {
			Amount           *int64  `json:"amount"`
			ReverseAmount    *int64  `json:"reverse_amount"`
			NetAmount        *int64  `json:"net_amount"`
			Status           *string `json:"status"`
			PaymentReference *string `json:"payment_reference"`
			Reason           string  `json:"reason"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		item, err := s.repo.AdjustSettlement(r.Context(), id, AdjustSettlementInput{
			Amount: req.Amount, ReverseAmount: req.ReverseAmount, NetAmount: req.NetAmount,
			Status: req.Status, PaymentReference: req.PaymentReference, Reason: req.Reason,
			OperatorID: subject.User.ID,
		})
		writeResult(w, item, err)
		return
	}
	if path == "/agent-audit-logs" && r.Method == http.MethodGet {
		filter := parseListFilter(r)
		items, total, err := s.repo.ListAuditLogs(r.Context(), filter)
		writePage(w, items, total, filter, err)
		return
	}

	WriteError(w, http.StatusNotFound, 404, "not found")
}

func (s *Server) routeAdminAgent(w http.ResponseWriter, r *http.Request, rest string, subject Subject) {
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		WriteError(w, http.StatusNotFound, 404, "not found")
		return
	}
	id, ok := parseID(w, parts[0])
	if !ok {
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			agent, err := s.repo.GetAgentByID(r.Context(), id)
			writeResult(w, agent, err)
		case http.MethodPut:
			var req struct {
				Level         *int   `json:"level"`
				ParentAgentID *int64 `json:"parent_agent_id"`
				RateBPS       *int   `json:"rate_bps"`
			}
			if !decodeJSON(w, r, &req) {
				return
			}
			agent, err := s.repo.UpdateAgent(r.Context(), id, UpdateAgentInput{Level: req.Level, ParentAgentID: req.ParentAgentID, RateBPS: req.RateBPS, OperatorID: subject.User.ID})
			writeResult(w, agent, err)
		default:
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
		}
		return
	}

	switch parts[1] {
	case "disable":
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}
		agent, err := s.repo.SetAgentStatus(r.Context(), id, "disabled", subject.User.ID)
		writeResult(w, agent, err)
	case "restore":
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}
		agent, err := s.repo.SetAgentStatus(r.Context(), id, "active", subject.User.ID)
		writeResult(w, agent, err)
	case "customers":
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}
		filter := parseListFilter(r)
		filter.AgentID = &id
		items, total, err := s.repo.ListCustomers(r.Context(), filter)
		writePage(w, items, total, filter, err)
	case "children":
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}
		filter := parseListFilter(r)
		items, total, err := s.repo.ListChildren(r.Context(), id, filter)
		writePage(w, items, total, filter, err)
	case "summary":
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}
		summary, err := s.repo.AgentSummary(r.Context(), id)
		writeResult(w, summary, err)
	case "commission-rate":
		if r.Method != http.MethodPut {
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}
		var req struct {
			RateBPS int `json:"rate_bps"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		agent, err := s.repo.UpdateAgent(r.Context(), id, UpdateAgentInput{RateBPS: &req.RateBPS, OperatorID: subject.User.ID})
		writeResult(w, agent, err)
	case "force-adjust":
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}
		var req struct {
			Level         *int   `json:"level"`
			ParentAgentID *int64 `json:"parent_agent_id"`
			RateBPS       *int   `json:"rate_bps"`
			Reason        string `json:"reason"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		agent, err := s.repo.UpdateAgent(r.Context(), id, UpdateAgentInput{Level: req.Level, ParentAgentID: req.ParentAgentID, RateBPS: req.RateBPS, OperatorID: subject.User.ID, Force: true, Reason: req.Reason})
		writeResult(w, agent, err)
	default:
		WriteError(w, http.StatusNotFound, 404, "not found")
	}
}

func (s *Server) routeAgent(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agent")
	subject, _ := SubjectFromContext(r.Context())
	agentID := subject.Agent.ID

	switch {
	case path == "/profile" && r.Method == http.MethodGet:
		writeResult(w, subject.Agent, nil)
	case path == "/dashboard" && r.Method == http.MethodGet:
		summary, err := s.repo.AgentSummary(r.Context(), agentID)
		writeResult(w, summary, err)
	case path == "/customers" && r.Method == http.MethodGet:
		filter := parseListFilter(r)
		filter.AgentID = &agentID
		items, total, err := s.repo.ListCustomers(r.Context(), filter)
		writePage(w, items, total, filter, err)
	case path == "/developable-users" && r.Method == http.MethodGet:
		filter := parseListFilter(r)
		items, total, err := s.repo.ListDevelopableUsers(r.Context(), *subject.Agent, filter)
		writePage(w, items, total, filter, err)
	case path == "/invites" && r.Method == http.MethodGet:
		filter := parseListFilter(r)
		items, total, err := s.repo.ListDevelopableUsers(r.Context(), *subject.Agent, filter)
		writePage(w, items, total, filter, err)
	case path == "/children" && r.Method == http.MethodGet:
		filter := parseListFilter(r)
		items, total, err := s.repo.ListChildren(r.Context(), agentID, filter)
		writePage(w, items, total, filter, err)
	case path == "/children" && r.Method == http.MethodPost:
		var req struct {
			UserID  int64 `json:"user_id"`
			RateBPS *int  `json:"rate_bps"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		agent, err := s.repo.CreateChildAgent(r.Context(), *subject.Agent, req.UserID, req.RateBPS, subject.User.ID)
		writeResult(w, agent, err)
	case strings.HasPrefix(path, "/children/") && strings.HasSuffix(path, "/commission-rate") && r.Method == http.MethodPut:
		idText := strings.TrimSuffix(strings.TrimPrefix(path, "/children/"), "/commission-rate")
		id, ok := parseID(w, idText)
		if !ok {
			return
		}
		var req struct {
			RateBPS int `json:"rate_bps"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		agent, err := s.repo.UpdateChildRate(r.Context(), *subject.Agent, id, req.RateBPS, subject.User.ID)
		writeResult(w, agent, err)
	case path == "/commissions" && r.Method == http.MethodGet:
		filter := parseListFilter(r)
		filter.AgentID = &agentID
		items, total, err := s.repo.ListCommissions(r.Context(), filter)
		writePage(w, items, total, filter, err)
	case path == "/settlements" && r.Method == http.MethodGet:
		filter := parseListFilter(r)
		filter.AgentID = &agentID
		items, total, err := s.repo.ListSettlements(r.Context(), filter)
		writePage(w, items, total, filter, err)
	case path == "/upline" && r.Method == http.MethodGet:
		if subject.Agent.ParentAgentID == nil {
			WriteOK(w, nil)
			return
		}
		agent, err := s.repo.GetAgentByID(r.Context(), *subject.Agent.ParentAgentID)
		writeResult(w, agent, err)
	case path == "/orders" && r.Method == http.MethodGet:
		filter := parseListFilter(r)
		items, total, err := s.repo.ListOrders(r.Context(), agentID, filter)
		writePage(w, items, total, filter, err)
	default:
		WriteError(w, http.StatusNotFound, 404, "not found")
	}
}

func (s *Server) shouldProxyToSub2API(path string) bool {
	for _, prefix := range s.cfg.Sub2APIProxyPrefixes {
		if prefix != "" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func newSub2APIProxy(rawBaseURL string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Set("X-Forwarded-Host", req.Host)
		if req.Header.Get("X-Forwarded-Proto") == "" {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
	}
	return proxy, nil
}

func parseListFilter(r *http.Request) ListFilter {
	q := r.URL.Query()
	return ListFilter{
		Page:     intQuery(q.Get("page"), 1),
		PageSize: intQuery(q.Get("page_size"), 20),
		Search:   strings.TrimSpace(q.Get("search")),
	}
}

func intQuery(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func parseID(w http.ResponseWriter, value string) (int64, bool) {
	id, err := strconv.ParseInt(strings.Trim(value, "/"), 10, 64)
	if err != nil || id <= 0 {
		WriteError(w, http.StatusBadRequest, 400, "invalid id")
		return 0, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid json")
		return false
	}
	return true
}

func writeResult(w http.ResponseWriter, data any, err error) {
	if err != nil {
		writeAppError(w, err)
		return
	}
	WriteOK(w, data)
}

func writePage[T any](w http.ResponseWriter, items []T, total int64, filter ListFilter, err error) {
	if err != nil {
		writeAppError(w, err)
		return
	}
	limit := filter.Limit()
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pages := 0
	if total > 0 {
		pages = int((total + int64(limit) - 1) / int64(limit))
	}
	WriteOK(w, Page[T]{Items: items, Total: total, Page: page, PageSize: limit, Pages: pages})
}

func writeAppError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, 404, "not found")
		return
	}
	if err == ErrForbidden {
		WriteError(w, http.StatusForbidden, 403, "forbidden")
		return
	}
	message := err.Error()
	status := http.StatusBadRequest
	if strings.Contains(message, "database") || strings.Contains(message, "query") {
		status = http.StatusInternalServerError
	}
	WriteError(w, status, status, fmt.Sprintf("%s", message))
}
