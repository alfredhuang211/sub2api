package app

import "time"

type User struct {
	ID           int64
	Email        string
	Username     string
	Role         string
	Status       string
	PasswordHash string
	TokenVersion int64
}

type UserOption struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type CurrentUser struct {
	ID          int64         `json:"id"`
	Email       string        `json:"email"`
	Username    string        `json:"username"`
	Role        string        `json:"role"`
	Status      string        `json:"status"`
	IsBaseAdmin bool          `json:"is_base_admin"`
	IsAdmin     bool          `json:"is_admin"`
	IsAgent     bool          `json:"is_agent"`
	Agent       *AgentProfile `json:"agent,omitempty"`
}

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
	ChildrenCount  int64      `json:"children_count"`
	CustomersCount int64      `json:"customers_count"`
	PayableAmount  int64      `json:"payable_amount"`
	FrozenAmount   int64      `json:"frozen_amount"`
	CreatedAt      time.Time  `json:"created_at"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
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

type AgentCustomerRelationChange struct {
	ID             int64     `json:"id"`
	CustomerUserID int64     `json:"customer_user_id"`
	CustomerEmail  string    `json:"customer_email"`
	FromAgentID    *int64    `json:"from_agent_id,omitempty"`
	FromAgentEmail *string   `json:"from_agent_email,omitempty"`
	ToAgentID      *int64    `json:"to_agent_id,omitempty"`
	ToAgentEmail   *string   `json:"to_agent_email,omitempty"`
	Reason         string    `json:"reason"`
	OperatorEmail  string    `json:"operator_email"`
	EffectiveAt    time.Time `json:"effective_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type AgentCommission struct {
	ID                int64     `json:"id"`
	CustomerEmail     string    `json:"customer_email"`
	OrderID           *int64    `json:"order_id,omitempty"`
	PeriodStartAt     time.Time `json:"period_start_at"`
	PeriodEndAt       time.Time `json:"period_end_at"`
	OrderPaidAmount   int64     `json:"order_paid_amount"`
	ConfirmedRevenue  int64     `json:"confirmed_revenue"`
	RateBPS           int       `json:"rate_bps"`
	CommissionAmount  int64     `json:"commission_amount"`
	ReverseAmount     int64     `json:"reverse_amount"`
	ReverseReasonType *string   `json:"reverse_reason_type,omitempty"`
	Status            string    `json:"status"`
}

type AgentSettlement struct {
	ID                   int64      `json:"id"`
	AgentID              int64      `json:"agent_id"`
	AgentEmail           string     `json:"agent_email"`
	PeriodMonth          string     `json:"period_month"`
	Amount               int64      `json:"amount"`
	ReverseAmount        int64      `json:"reverse_amount"`
	NetAmount            int64      `json:"net_amount"`
	Status               string     `json:"status"`
	FrozenUntil          *time.Time `json:"frozen_until,omitempty"`
	PaidAt               *time.Time `json:"paid_at,omitempty"`
	PaymentAmount        *int64     `json:"payment_amount,omitempty"`
	PaymentMethod        *string    `json:"payment_method,omitempty"`
	PaymentReference     *string    `json:"payment_reference,omitempty"`
	PaymentRemark        *string    `json:"payment_remark,omitempty"`
	PaymentRegisteredAt  *time.Time `json:"payment_registered_at,omitempty"`
	PaymentOperatorEmail *string    `json:"payment_operator_email,omitempty"`
}

type AgentAuditLog struct {
	ID            int64     `json:"id"`
	OperatorEmail string    `json:"operator_email"`
	OperatorRole  string    `json:"operator_role"`
	Action        string    `json:"action"`
	TargetType    string    `json:"target_type"`
	TargetID      int64     `json:"target_id"`
	Reason        *string   `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AgentAdminSettings struct {
	TurnstileEnabled bool      `json:"turnstile_enabled"`
	TurnstileSiteKey string    `json:"turnstile_site_key"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type AgentAdminUser struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	Email          string     `json:"email"`
	Username       string     `json:"username"`
	Source         string     `json:"source"`
	Status         string     `json:"status"`
	CreatedByEmail *string    `json:"created_by_email,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

type PublicSettings struct {
	TurnstileEnabled bool   `json:"turnstile_enabled"`
	TurnstileSiteKey string `json:"turnstile_site_key"`
}

type AgentOrder struct {
	ID            int64      `json:"id"`
	OrderNo       string     `json:"order_no"`
	CustomerEmail string     `json:"customer_email"`
	Status        string     `json:"status"`
	PayAmount     int64      `json:"pay_amount"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type ListFilter struct {
	Page     int
	PageSize int
	Search   string
	AgentID  *int64
}

func (f ListFilter) Limit() int {
	if f.PageSize <= 0 {
		return 20
	}
	if f.PageSize > 100 {
		return 100
	}
	return f.PageSize
}

func (f ListFilter) Offset() int {
	page := f.Page
	if page <= 0 {
		page = 1
	}
	return (page - 1) * f.Limit()
}
