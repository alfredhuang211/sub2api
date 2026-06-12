package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	AgentStatusActive   = "active"
	AgentStatusDisabled = "disabled"

	AgentSourceReferral = "referral"
	AgentSourceManual   = "manual"

	defaultAgentLevel1RateBPS = 2000
	defaultAgentLevel2RateBPS = 1500
	defaultAgentLevel3RateBPS = 1000
)

var (
	ErrAgentNotFound        = infraerrors.NotFound("AGENT_NOT_FOUND", "agent profile not found")
	ErrAgentDisabled        = infraerrors.Forbidden("AGENT_DISABLED", "agent is disabled")
	ErrAgentInvalidLevel    = infraerrors.BadRequest("AGENT_INVALID_LEVEL", "agent level must be 1, 2 or 3")
	ErrAgentParentRequired  = infraerrors.BadRequest("AGENT_PARENT_REQUIRED", "parent agent is required for level 2 or 3")
	ErrAgentParentInvalid   = infraerrors.BadRequest("AGENT_PARENT_INVALID", "parent agent level is invalid")
	ErrAgentRateInvalid     = infraerrors.BadRequest("AGENT_RATE_INVALID", "commission rate is invalid")
	ErrAgentCustomerInvalid = infraerrors.BadRequest("AGENT_CUSTOMER_INVALID", "agent cannot be assigned to this customer")
	ErrAgentAlreadyExists   = infraerrors.Conflict("AGENT_ALREADY_EXISTS", "user is already an agent")
)

type AgentProfile struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	Level          int        `json:"level"`
	ParentAgentID  *int64     `json:"parent_agent_id,omitempty"`
	ParentUsername *string    `json:"parent_username,omitempty"`
	ParentEmail    *string    `json:"parent_email,omitempty"`
	Status         string     `json:"status"`
	RateBPS        int        `json:"rate_bps"`
	ChildrenCount  int        `json:"children_count"`
	CustomersCount int        `json:"customers_count"`
	PayableAmount  int64      `json:"payable_amount"`
	FrozenAmount   int64      `json:"frozen_amount"`
	CreatedAt      time.Time  `json:"created_at"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
	ContactInfo    any        `json:"contact_info,omitempty"`
	SettlementInfo any        `json:"settlement_info,omitempty"`
}

type AgentSummary struct {
	TotalAgents      int64 `json:"total_agents"`
	ActiveAgents     int64 `json:"active_agents"`
	DisabledAgents   int64 `json:"disabled_agents"`
	DirectCustomers  int64 `json:"direct_customers"`
	ChildAgents      int64 `json:"child_agents"`
	ConfirmedRevenue int64 `json:"confirmed_revenue"`
	CommissionAmount int64 `json:"commission_amount"`
	PayableAmount    int64 `json:"payable_amount"`
	ReversedAmount   int64 `json:"reversed_amount"`
}

type AgentCustomer struct {
	ID                 int64      `json:"id"`
	UserID             int64      `json:"user_id"`
	Email              string     `json:"email"`
	Username           string     `json:"username"`
	Source             string     `json:"source"`
	SourceReferralCode *string    `json:"source_referral_code,omitempty"`
	AgentID            int64      `json:"agent_id"`
	AgentName          string     `json:"agent_name"`
	SubscriptionName   *string    `json:"subscription_name,omitempty"`
	PeriodEndAt        *time.Time `json:"period_end_at,omitempty"`
	ConfirmedRevenue   int64      `json:"confirmed_revenue"`
	Status             string     `json:"status"`
}

type AgentCommission struct {
	ID                int64      `json:"id"`
	CustomerEmail     string     `json:"customer_email"`
	OrderID           *int64     `json:"order_id,omitempty"`
	PeriodStartAt     time.Time  `json:"period_start_at"`
	PeriodEndAt       time.Time  `json:"period_end_at"`
	OrderPaidAmount   int64      `json:"order_paid_amount"`
	ConfirmedRevenue  int64      `json:"confirmed_revenue"`
	RateBPS           int        `json:"rate_bps"`
	CommissionAmount  int64      `json:"commission_amount"`
	ReverseAmount     int64      `json:"reverse_amount"`
	ReverseReasonType *string    `json:"reverse_reason_type,omitempty"`
	Status            string     `json:"status"`
	FrozenUntil       *time.Time `json:"frozen_until,omitempty"`
}

type AgentSettlement struct {
	ID               int64      `json:"id"`
	AgentID          int64      `json:"agent_id"`
	AgentEmail       string     `json:"agent_email"`
	PeriodMonth      string     `json:"period_month"`
	Amount           int64      `json:"amount"`
	ReverseAmount    int64      `json:"reverse_amount"`
	NetAmount        int64      `json:"net_amount"`
	Status           string     `json:"status"`
	FrozenUntil      *time.Time `json:"frozen_until,omitempty"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	PaymentReference *string    `json:"payment_reference,omitempty"`
}

type AgentAuditLog struct {
	ID            int64     `json:"id"`
	OperatorID    *int64    `json:"operator_user_id,omitempty"`
	OperatorEmail string    `json:"operator_email"`
	OperatorRole  string    `json:"operator_role"`
	Action        string    `json:"action"`
	TargetType    string    `json:"target_type"`
	TargetID      int64     `json:"target_id"`
	Reason        *string   `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AgentListFilter struct {
	Search   string
	Page     int
	PageSize int
	AgentID  *int64
}

type CreateAgentInput struct {
	UserID        int64
	Level         int
	ParentAgentID *int64
	RateBPS       *int
	OperatorID    int64
}

type UpdateAgentInput struct {
	Level         *int
	ParentAgentID *int64
	RateBPS       *int
	OperatorID    int64
}

type AssignAgentCustomerInput struct {
	CustomerUserID int64
	AgentID        int64
	Reason         string
	OperatorID     int64
}

type AgentAuditInput struct {
	OperatorUserID *int64
	OperatorRole   string
	Action         string
	TargetType     string
	TargetID       int64
	BeforeData     any
	AfterData      any
	Reason         *string
}

type AgentRepository interface {
	GetAgent(ctx context.Context, id int64) (*AgentProfile, error)
	GetAgentByUserID(ctx context.Context, userID int64) (*AgentProfile, error)
	ListAgents(ctx context.Context, filter AgentListFilter) ([]AgentProfile, int64, error)
	CreateAgent(ctx context.Context, input CreateAgentInput, rateBPS int) (*AgentProfile, error)
	UpdateAgent(ctx context.Context, id int64, input UpdateAgentInput, rateBPS *int) (*AgentProfile, error)
	SetAgentStatus(ctx context.Context, id int64, status string, operatorID int64) (*AgentProfile, error)
	AssignCustomer(ctx context.Context, input AssignAgentCustomerInput) (*AgentCustomer, error)
	ListCustomers(ctx context.Context, filter AgentListFilter) ([]AgentCustomer, int64, error)
	ListCommissions(ctx context.Context, filter AgentListFilter) ([]AgentCommission, int64, error)
	ListSettlements(ctx context.Context, filter AgentListFilter) ([]AgentSettlement, int64, error)
	MarkSettlementPaid(ctx context.Context, id, operatorID int64) (*AgentSettlement, error)
	ListAuditLogs(ctx context.Context, filter AgentListFilter) ([]AgentAuditLog, int64, error)
	CreateAuditLog(ctx context.Context, input AgentAuditInput) error
	GetSummary(ctx context.Context, agentID *int64) (*AgentSummary, error)
}

type AgentService struct {
	repo AgentRepository
}

func NewAgentService(repo AgentRepository) *AgentService {
	return &AgentService{repo: repo}
}

func (s *AgentService) AdminSummary(ctx context.Context) (*AgentSummary, error) {
	return s.repo.GetSummary(ctx, nil)
}

func (s *AgentService) ListAgents(ctx context.Context, filter AgentListFilter) ([]AgentProfile, int64, error) {
	normalizeAgentListFilter(&filter)
	return s.repo.ListAgents(ctx, filter)
}

func (s *AgentService) GetAgent(ctx context.Context, id int64) (*AgentProfile, error) {
	if id <= 0 {
		return nil, ErrAgentNotFound
	}
	return s.repo.GetAgent(ctx, id)
}

func (s *AgentService) CreateAgent(ctx context.Context, input CreateAgentInput) (*AgentProfile, error) {
	if input.UserID <= 0 {
		return nil, ErrUserNotFound
	}
	if err := validateAgentLevel(input.Level); err != nil {
		return nil, err
	}
	rate := defaultAgentRateBPS(input.Level)
	if input.RateBPS != nil {
		rate = *input.RateBPS
	}
	if err := s.validateParentAndRate(ctx, input.Level, input.ParentAgentID, rate); err != nil {
		return nil, err
	}
	agent, err := s.repo.CreateAgent(ctx, input, rate)
	if err != nil {
		return nil, err
	}
	_ = s.repo.CreateAuditLog(ctx, AgentAuditInput{
		OperatorUserID: &input.OperatorID,
		OperatorRole:   RoleAdmin,
		Action:         "agent.create",
		TargetType:     "agent_profile",
		TargetID:       agent.ID,
		AfterData:      agent,
	})
	return agent, nil
}

func (s *AgentService) UpdateAgent(ctx context.Context, id int64, input UpdateAgentInput) (*AgentProfile, error) {
	current, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}
	level := current.Level
	if input.Level != nil {
		level = *input.Level
	}
	if err := validateAgentLevel(level); err != nil {
		return nil, err
	}
	parentID := current.ParentAgentID
	if input.ParentAgentID != nil || level == 1 {
		parentID = input.ParentAgentID
	}
	rate := current.RateBPS
	if input.RateBPS != nil {
		rate = *input.RateBPS
	}
	if err := s.validateParentAndRate(ctx, level, parentID, rate); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateAgent(ctx, id, input, input.RateBPS)
	if err != nil {
		return nil, err
	}
	_ = s.repo.CreateAuditLog(ctx, AgentAuditInput{
		OperatorUserID: &input.OperatorID,
		OperatorRole:   RoleAdmin,
		Action:         "agent.update",
		TargetType:     "agent_profile",
		TargetID:       id,
		BeforeData:     current,
		AfterData:      updated,
	})
	return updated, nil
}

func (s *AgentService) DisableAgent(ctx context.Context, id, operatorID int64) (*AgentProfile, error) {
	return s.setStatus(ctx, id, AgentStatusDisabled, operatorID, "agent.disable")
}

func (s *AgentService) RestoreAgent(ctx context.Context, id, operatorID int64) (*AgentProfile, error) {
	return s.setStatus(ctx, id, AgentStatusActive, operatorID, "agent.restore")
}

func (s *AgentService) AssignCustomer(ctx context.Context, input AssignAgentCustomerInput) (*AgentCustomer, error) {
	if input.CustomerUserID <= 0 || input.AgentID <= 0 {
		return nil, ErrAgentCustomerInvalid
	}
	if strings.TrimSpace(input.Reason) == "" {
		return nil, infraerrors.BadRequest("AGENT_ASSIGN_REASON_REQUIRED", "reason is required")
	}
	agent, err := s.repo.GetAgent(ctx, input.AgentID)
	if err != nil {
		return nil, err
	}
	if agent.Status != AgentStatusActive {
		return nil, ErrAgentDisabled
	}
	if agent.UserID == input.CustomerUserID {
		return nil, ErrAgentCustomerInvalid
	}
	customer, err := s.repo.AssignCustomer(ctx, input)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(input.Reason)
	_ = s.repo.CreateAuditLog(ctx, AgentAuditInput{
		OperatorUserID: &input.OperatorID,
		OperatorRole:   RoleAdmin,
		Action:         "customer.assign",
		TargetType:     "agent_customer_relation",
		TargetID:       customer.ID,
		AfterData:      customer,
		Reason:         &reason,
	})
	return customer, nil
}

func (s *AgentService) ListCustomers(ctx context.Context, filter AgentListFilter) ([]AgentCustomer, int64, error) {
	normalizeAgentListFilter(&filter)
	return s.repo.ListCustomers(ctx, filter)
}

func (s *AgentService) ListCommissions(ctx context.Context, filter AgentListFilter) ([]AgentCommission, int64, error) {
	normalizeAgentListFilter(&filter)
	return s.repo.ListCommissions(ctx, filter)
}

func (s *AgentService) ListSettlements(ctx context.Context, filter AgentListFilter) ([]AgentSettlement, int64, error) {
	normalizeAgentListFilter(&filter)
	return s.repo.ListSettlements(ctx, filter)
}

func (s *AgentService) MarkSettlementPaid(ctx context.Context, id, operatorID int64) (*AgentSettlement, error) {
	settlement, err := s.repo.MarkSettlementPaid(ctx, id, operatorID)
	if err != nil {
		return nil, err
	}
	_ = s.repo.CreateAuditLog(ctx, AgentAuditInput{
		OperatorUserID: &operatorID,
		OperatorRole:   RoleAdmin,
		Action:         "settlement.mark_paid",
		TargetType:     "agent_settlement",
		TargetID:       id,
		AfterData:      settlement,
	})
	return settlement, nil
}

func (s *AgentService) ListAuditLogs(ctx context.Context, filter AgentListFilter) ([]AgentAuditLog, int64, error) {
	normalizeAgentListFilter(&filter)
	return s.repo.ListAuditLogs(ctx, filter)
}

func (s *AgentService) GetMyProfile(ctx context.Context, userID int64) (*AgentProfile, error) {
	agent, err := s.repo.GetAgentByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if agent.Status != AgentStatusActive {
		return nil, ErrAgentDisabled
	}
	return agent, nil
}

func (s *AgentService) GetMySummary(ctx context.Context, userID int64) (*AgentSummary, error) {
	agent, err := s.GetMyProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetSummary(ctx, &agent.ID)
}

func (s *AgentService) GetMyUpline(ctx context.Context, userID int64) (*AgentProfile, error) {
	agent, err := s.GetMyProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if agent.ParentAgentID == nil {
		return nil, nil
	}
	return s.repo.GetAgent(ctx, *agent.ParentAgentID)
}

func (s *AgentService) MyCustomers(ctx context.Context, userID int64, filter AgentListFilter) ([]AgentCustomer, int64, error) {
	agent, err := s.GetMyProfile(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	filter.AgentID = &agent.ID
	return s.ListCustomers(ctx, filter)
}

func (s *AgentService) MyCommissions(ctx context.Context, userID int64, filter AgentListFilter) ([]AgentCommission, int64, error) {
	agent, err := s.GetMyProfile(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	filter.AgentID = &agent.ID
	return s.ListCommissions(ctx, filter)
}

func (s *AgentService) MySettlements(ctx context.Context, userID int64, filter AgentListFilter) ([]AgentSettlement, int64, error) {
	agent, err := s.GetMyProfile(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	filter.AgentID = &agent.ID
	return s.ListSettlements(ctx, filter)
}

func (s *AgentService) setStatus(ctx context.Context, id int64, status string, operatorID int64, action string) (*AgentProfile, error) {
	current, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.SetAgentStatus(ctx, id, status, operatorID)
	if err != nil {
		return nil, err
	}
	_ = s.repo.CreateAuditLog(ctx, AgentAuditInput{
		OperatorUserID: &operatorID,
		OperatorRole:   RoleAdmin,
		Action:         action,
		TargetType:     "agent_profile",
		TargetID:       id,
		BeforeData:     current,
		AfterData:      updated,
	})
	return updated, nil
}

func (s *AgentService) validateParentAndRate(ctx context.Context, level int, parentID *int64, rateBPS int) error {
	if rateBPS < 0 || rateBPS > 10000 {
		return ErrAgentRateInvalid
	}
	if level == 1 {
		if parentID != nil {
			return ErrAgentParentInvalid
		}
		return nil
	}
	if parentID == nil || *parentID <= 0 {
		return ErrAgentParentRequired
	}
	parent, err := s.repo.GetAgent(ctx, *parentID)
	if err != nil {
		return err
	}
	if parent.Status != AgentStatusActive {
		return ErrAgentParentInvalid
	}
	if parent.Level != level-1 {
		return ErrAgentParentInvalid
	}
	if rateBPS >= parent.RateBPS {
		return infraerrors.BadRequest("AGENT_RATE_MUST_BE_LOWER_THAN_PARENT", "child agent rate must be lower than parent agent rate")
	}
	return nil
}

func validateAgentLevel(level int) error {
	if level < 1 || level > 3 {
		return ErrAgentInvalidLevel
	}
	return nil
}

func defaultAgentRateBPS(level int) int {
	switch level {
	case 1:
		return defaultAgentLevel1RateBPS
	case 2:
		return defaultAgentLevel2RateBPS
	case 3:
		return defaultAgentLevel3RateBPS
	default:
		return 0
	}
}

func normalizeAgentListFilter(filter *AgentListFilter) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	filter.Search = strings.TrimSpace(filter.Search)
}
