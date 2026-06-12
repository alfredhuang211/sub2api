package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	agentService *service.AgentService
}

func NewAgentHandler(agentService *service.AgentService) *AgentHandler {
	return &AgentHandler{agentService: agentService}
}

type CreateAgentRequest struct {
	UserID        int64  `json:"user_id" binding:"required,gt=0"`
	Level         int    `json:"level" binding:"required"`
	ParentAgentID *int64 `json:"parent_agent_id"`
	RateBPS       *int   `json:"rate_bps"`
}

type UpdateAgentRequest struct {
	Level         *int   `json:"level"`
	ParentAgentID *int64 `json:"parent_agent_id"`
	RateBPS       *int   `json:"rate_bps"`
}

type AssignCustomerRequest struct {
	CustomerUserID int64  `json:"customer_user_id" binding:"required,gt=0"`
	AgentID        int64  `json:"agent_id" binding:"required,gt=0"`
	Reason         string `json:"reason" binding:"required"`
}

func (h *AgentHandler) Summary(c *gin.Context) {
	summary, err := h.agentService.AdminSummary(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *AgentHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.agentService.ListAgents(c.Request.Context(), service.AgentListFilter{
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *AgentHandler) Get(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	agent, err := h.agentService.GetAgent(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agent)
}

func (h *AgentHandler) Create(c *gin.Context) {
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	agent, err := h.agentService.CreateAgent(c.Request.Context(), service.CreateAgentInput{
		UserID:        req.UserID,
		Level:         req.Level,
		ParentAgentID: req.ParentAgentID,
		RateBPS:       req.RateBPS,
		OperatorID:    subject.UserID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agent)
}

func (h *AgentHandler) Update(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	agent, err := h.agentService.UpdateAgent(c.Request.Context(), id, service.UpdateAgentInput{
		Level:         req.Level,
		ParentAgentID: req.ParentAgentID,
		RateBPS:       req.RateBPS,
		OperatorID:    subject.UserID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agent)
}

func (h *AgentHandler) Disable(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	agent, err := h.agentService.DisableAgent(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agent)
}

func (h *AgentHandler) Restore(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	agent, err := h.agentService.RestoreAgent(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agent)
}

func (h *AgentHandler) ListCustomers(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.agentService.ListCustomers(c.Request.Context(), service.AgentListFilter{
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
		AgentID:  &id,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *AgentHandler) AssignCustomer(c *gin.Context) {
	var req AssignCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	customer, err := h.agentService.AssignCustomer(c.Request.Context(), service.AssignAgentCustomerInput{
		CustomerUserID: req.CustomerUserID,
		AgentID:        req.AgentID,
		Reason:         req.Reason,
		OperatorID:     subject.UserID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, customer)
}

func (h *AgentHandler) ListCommissions(c *gin.Context) {
	h.listCommissions(c, nil)
}

func (h *AgentHandler) ListSettlements(c *gin.Context) {
	h.listSettlements(c, nil)
}

func (h *AgentHandler) MarkSettlementPaid(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	settlement, err := h.agentService.MarkSettlementPaid(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settlement)
}

func (h *AgentHandler) ListAuditLogs(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.agentService.ListAuditLogs(c.Request.Context(), service.AgentListFilter{
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *AgentHandler) listCommissions(c *gin.Context, agentID *int64) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.agentService.ListCommissions(c.Request.Context(), service.AgentListFilter{
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
		AgentID:  agentID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *AgentHandler) listSettlements(c *gin.Context, agentID *int64) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.agentService.ListSettlements(c.Request.Context(), service.AgentListFilter{
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
		AgentID:  agentID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func parsePositiveIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return id, true
}
