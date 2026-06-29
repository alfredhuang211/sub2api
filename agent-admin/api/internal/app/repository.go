package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.db.QueryRowContext(ctx, `
SELECT id, email, COALESCE(username, ''), role, status, password_hash
FROM users
WHERE id = $1 AND deleted_at IS NULL`, id).Scan(
		&u.ID, &u.Email, &u.Username, &u.Role, &u.Status, &u.PasswordHash,
	)
	return u, err
}

func (r *Repository) SearchAssignableUsers(ctx context.Context, filter ListFilter) ([]UserOption, int64, error) {
	where := ` WHERE u.deleted_at IS NULL AND ap.id IS NULL`
	args := []any{}
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.username ILIKE $%d)", len(args), len(args))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM users u
LEFT JOIN agent_profiles ap ON ap.user_id = u.id
`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT u.id, u.email, COALESCE(u.username, ''), u.role, u.status
FROM users u
LEFT JOIN agent_profiles ap ON ap.user_id = u.id
`+where+` ORDER BY u.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []UserOption{}
	for rows.Next() {
		var item UserOption
		if err := rows.Scan(&item.ID, &item.Email, &item.Username, &item.Role, &item.Status); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) SearchUsers(ctx context.Context, filter ListFilter) ([]UserOption, int64, error) {
	where := ` WHERE u.deleted_at IS NULL`
	args := []any{}
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.username ILIKE $%d)", len(args), len(args))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users u`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT u.id, u.email, COALESCE(u.username, ''), u.role, u.status
FROM users u
`+where+` ORDER BY u.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []UserOption{}
	for rows.Next() {
		var item UserOption
		if err := rows.Scan(&item.ID, &item.Email, &item.Username, &item.Role, &item.Status); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) SearchNonAdminUsers(ctx context.Context, filter ListFilter) ([]UserOption, int64, error) {
	where := ` WHERE u.deleted_at IS NULL AND u.status = 'active' AND u.role <> 'admin' AND active_admin.user_id IS NULL`
	args := []any{}
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.username ILIKE $%d)", len(args), len(args))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM users u
LEFT JOIN agent_admin_users active_admin ON active_admin.user_id = u.id AND active_admin.status = 'active'
`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT u.id, u.email, COALESCE(u.username, ''), u.role, u.status
FROM users u
LEFT JOIN agent_admin_users active_admin ON active_admin.user_id = u.id AND active_admin.status = 'active'
`+where+` ORDER BY u.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []UserOption{}
	for rows.Next() {
		var item UserOption
		if err := rows.Scan(&item.ID, &item.Email, &item.Username, &item.Role, &item.Status); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetAgentByUserID(ctx context.Context, userID int64) (*AgentProfile, error) {
	rows, err := r.db.QueryContext(ctx, agentSelectSQL()+` WHERE ap.user_id = $1 AND u.deleted_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOneAgent(rows)
}

func (r *Repository) GetAgentByID(ctx context.Context, id int64) (*AgentProfile, error) {
	rows, err := r.db.QueryContext(ctx, agentSelectSQL()+` WHERE ap.id = $1 AND u.deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOneAgent(rows)
}

func (r *Repository) AdminSummary(ctx context.Context) (AgentSummary, error) {
	var out AgentSummary
	err := r.db.QueryRowContext(ctx, `
SELECT
  COUNT(*)::bigint,
  COUNT(*) FILTER (WHERE ap.status = 'active')::bigint,
  COUNT(*) FILTER (WHERE ap.status = 'disabled')::bigint
FROM agent_profiles ap
JOIN users u ON u.id = ap.user_id AND u.deleted_at IS NULL`).Scan(&out.TotalAgents, &out.ActiveAgents, &out.DisabledAgents)
	if err != nil {
		return out, err
	}
	_ = r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::bigint
FROM agent_customer_relations acr
JOIN users u ON u.id = acr.customer_user_id AND u.deleted_at IS NULL
WHERE acr.status = 'active'`).Scan(&out.DirectCustomers)
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(confirmed_revenue),0)::bigint, COALESCE(SUM(commission_amount),0)::bigint, COALESCE(SUM(reverse_amount),0)::bigint FROM agent_commission_periods`).Scan(&out.ConfirmedRevenue, &out.CommissionAmount, &out.ReversedAmount)
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(net_amount),0)::bigint FROM agent_settlements WHERE status IN ('payable','pending')`).Scan(&out.PayableAmount)
	_ = r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::bigint
FROM agent_profiles ap
JOIN users u ON u.id = ap.user_id AND u.deleted_at IS NULL
WHERE ap.parent_agent_id IS NOT NULL`).Scan(&out.ChildAgents)
	return out, nil
}

func (r *Repository) AgentSummary(ctx context.Context, agentID int64) (AgentSummary, error) {
	var out AgentSummary
	_ = r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::bigint
FROM agent_customer_relations acr
JOIN users u ON u.id = acr.customer_user_id AND u.deleted_at IS NULL
WHERE acr.status = 'active' AND acr.agent_id = $1`, agentID).Scan(&out.DirectCustomers)
	_ = r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::bigint
FROM agent_profiles ap
JOIN users u ON u.id = ap.user_id AND u.deleted_at IS NULL
WHERE ap.parent_agent_id = $1`, agentID).Scan(&out.ChildAgents)
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(confirmed_revenue),0)::bigint, COALESCE(SUM(commission_amount),0)::bigint, COALESCE(SUM(reverse_amount),0)::bigint FROM agent_commission_periods WHERE agent_id = $1`, agentID).Scan(&out.ConfirmedRevenue, &out.CommissionAmount, &out.ReversedAmount)
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(net_amount),0)::bigint FROM agent_settlements WHERE agent_id = $1 AND status IN ('payable','pending')`, agentID).Scan(&out.PayableAmount)
	out.TotalAgents = 1
	out.ActiveAgents = 1
	return out, nil
}

func (r *Repository) ListAgents(ctx context.Context, filter ListFilter) ([]AgentProfile, int64, error) {
	where, args := buildAgentWhere(filter.Search)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_profiles ap JOIN users u ON u.id = ap.user_id `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, agentSelectSQL()+where+` ORDER BY ap.created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanAgents(rows)
	return items, total, err
}

type CreateAgentInput struct {
	UserID        int64
	Level         int
	ParentAgentID *int64
	RateBPS       *int
	OperatorID    int64
}

func (r *Repository) CreateAgent(ctx context.Context, in CreateAgentInput) (*AgentProfile, error) {
	rate := defaultRateBPS(in.Level)
	if in.RateBPS != nil {
		rate = *in.RateBPS
	}
	if err := r.validateAgentInput(ctx, in.Level, in.ParentAgentID, &rate); err != nil {
		return nil, err
	}
	if _, err := r.GetUserByID(ctx, in.UserID); err != nil {
		return nil, fmt.Errorf("user not found")
	}
	if _, err := r.GetAgentByUserID(ctx, in.UserID); err == nil {
		return nil, fmt.Errorf("user is already an agent")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var agentID int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO agent_profiles (user_id, level, parent_agent_id, status, created_by)
VALUES ($1, $2, $3, 'active', $4)
RETURNING id`, in.UserID, in.Level, in.ParentAgentID, in.OperatorID).Scan(&agentID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO agent_commission_rates (agent_id, rate_bps, set_by_user_id)
VALUES ($1, $2, $3)`, agentID, rate, in.OperatorID); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, in.OperatorID, "admin", "agent.create", "agent", agentID, nil, map[string]any{"rate_bps": rate}, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAgentByID(ctx, agentID)
}

type UpdateAgentInput struct {
	Level         *int
	ParentAgentID *int64
	RateBPS       *int
	OperatorID    int64
	Force         bool
	Reason        string
}

func (r *Repository) UpdateAgent(ctx context.Context, id int64, in UpdateAgentInput) (*AgentProfile, error) {
	current, err := r.GetAgentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	level := current.Level
	parentID := current.ParentAgentID
	if in.Level != nil {
		level = *in.Level
	}
	if in.ParentAgentID != nil {
		parentID = in.ParentAgentID
	}
	if level == 1 {
		parentID = nil
	}
	if parentID != nil && *parentID == id {
		return nil, fmt.Errorf("agent cannot be its own parent")
	}
	if parentID != nil {
		descendant, err := r.isDescendantAgent(ctx, *parentID, id)
		if err != nil {
			return nil, err
		}
		if descendant {
			return nil, fmt.Errorf("agent parent cannot be a descendant")
		}
	}
	rate := current.RateBPS
	if in.RateBPS != nil {
		rate = *in.RateBPS
	}
	if err := r.validateAgentInput(ctx, level, parentID, &rate); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE agent_profiles SET level=$1, parent_agent_id=$2, updated_at=NOW() WHERE id=$3`, level, parentID, id); err != nil {
		return nil, err
	}
	if in.RateBPS != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE agent_commission_rates SET expired_at=NOW(), updated_at=NOW() WHERE agent_id=$1 AND expired_at IS NULL`, id); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO agent_commission_rates (agent_id, rate_bps, set_by_user_id) VALUES ($1,$2,$3)`, id, *in.RateBPS, in.OperatorID); err != nil {
			return nil, err
		}
	}
	action := "agent.update"
	reason := (*string)(nil)
	if in.Force {
		action = "agent.force_adjust"
		trimmed := strings.TrimSpace(in.Reason)
		if trimmed == "" {
			return nil, fmt.Errorf("reason is required")
		}
		reason = &trimmed
	}
	if err := insertAudit(ctx, tx, in.OperatorID, "admin", action, "agent", id, current, map[string]any{"level": level, "parent_agent_id": parentID, "rate_bps": in.RateBPS}, reason); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAgentByID(ctx, id)
}

func (r *Repository) SetAgentStatus(ctx context.Context, id int64, status string, operatorID int64) (*AgentProfile, error) {
	if status != "active" && status != "disabled" {
		return nil, fmt.Errorf("invalid status")
	}
	disabledExpr := "NULL"
	if status == "disabled" {
		disabledExpr = "NOW()"
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE agent_profiles SET status=$1, disabled_at=`+disabledExpr+`, updated_at=NOW() WHERE id=$2`, status, id); err != nil {
		return nil, err
	}
	_ = r.InsertAudit(ctx, operatorID, "admin", "agent."+status, "agent", id, nil, nil)
	return r.GetAgentByID(ctx, id)
}

func (r *Repository) AssignCustomer(ctx context.Context, customerUserID, agentID, operatorID int64, reason string) (*AgentCustomer, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("reason is required")
	}
	if _, err := r.GetUserByID(ctx, customerUserID); err != nil {
		return nil, fmt.Errorf("customer user not found")
	}
	if _, err := r.GetAgentByID(ctx, agentID); err != nil {
		return nil, fmt.Errorf("agent not found")
	}
	effectiveAt, err := r.nextSubscriptionPeriodStart(ctx, customerUserID)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var fromAgentID sql.NullInt64
	_ = tx.QueryRowContext(ctx, `SELECT agent_id FROM agent_customer_relations WHERE customer_user_id=$1 AND status='active'`, customerUserID).Scan(&fromAgentID)
	if _, err := tx.ExecContext(ctx, `UPDATE agent_customer_relations SET status='expired', expired_at=NOW(), updated_at=NOW() WHERE customer_user_id=$1 AND status='scheduled'`, customerUserID); err != nil {
		return nil, err
	}
	var relationID int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO agent_customer_relations (agent_id, customer_user_id, source, effective_at, status, created_by)
VALUES ($1,$2,'manual',$3,'scheduled',$4)
RETURNING id`, agentID, customerUserID, effectiveAt, operatorID).Scan(&relationID); err != nil {
		return nil, err
	}
	var from *int64
	if fromAgentID.Valid {
		from = &fromAgentID.Int64
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO agent_customer_relation_changes (customer_user_id, from_agent_id, to_agent_id, reason, operator_user_id, effective_at)
VALUES ($1,$2,$3,$4,$5,$6)`, customerUserID, from, agentID, reason, operatorID, effectiveAt); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, operatorID, "admin", "customer.assign", "agent_customer_relation", relationID, map[string]any{"from_agent_id": from}, map[string]any{"to_agent_id": agentID, "effective_at": effectiveAt}, &reason); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetCustomerRelation(ctx, relationID)
}

func (r *Repository) nextSubscriptionPeriodStart(ctx context.Context, customerUserID int64) (time.Time, error) {
	var nextStart sql.NullTime
	err := r.db.QueryRowContext(ctx, `
SELECT MIN(expires_at)
FROM user_subscriptions
WHERE user_id = $1
  AND deleted_at IS NULL
  AND expires_at > NOW()`, customerUserID).Scan(&nextStart)
	if err != nil {
		return time.Time{}, err
	}
	if nextStart.Valid {
		return nextStart.Time, nil
	}
	return time.Now().UTC(), nil
}

func (r *Repository) ActivateScheduledCustomerRelations(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
WITH due AS (
    SELECT id, customer_user_id
    FROM agent_customer_relations
    WHERE status = 'scheduled'
      AND effective_at <= NOW()
),
expired AS (
    UPDATE agent_customer_relations acr
    SET status = 'expired',
        expired_at = NOW(),
        updated_at = NOW()
    FROM due
    WHERE acr.customer_user_id = due.customer_user_id
      AND acr.status = 'active'
      AND acr.id <> due.id
    RETURNING acr.id
)
UPDATE agent_customer_relations acr
SET status = 'active',
    updated_at = NOW()
FROM due
WHERE acr.id = due.id`); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) ListCustomers(ctx context.Context, filter ListFilter) ([]AgentCustomer, int64, error) {
	where := ` WHERE acr.status IN ('active', 'scheduled') AND u.deleted_at IS NULL`
	args := []any{}
	if filter.AgentID != nil {
		args = append(args, *filter.AgentID)
		where += fmt.Sprintf(" AND acr.agent_id = $%d", len(args))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		where += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.username ILIKE $%d)", len(args), len(args))
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_customer_relations acr JOIN users u ON u.id=acr.customer_user_id`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, customerSelectSQL()+where+` ORDER BY acr.created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanCustomers(rows)
	return items, total, err
}

func (r *Repository) ListCustomerRelationChanges(ctx context.Context, filter ListFilter) ([]AgentCustomerRelationChange, int64, error) {
	where := ""
	args := []any{}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		where = fmt.Sprintf(" WHERE customer.email ILIKE $%d OR COALESCE(operator.email,'') ILIKE $%d OR c.reason ILIKE $%d", len(args), len(args), len(args))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM agent_customer_relation_changes c
JOIN users customer ON customer.id = c.customer_user_id AND customer.deleted_at IS NULL
LEFT JOIN users operator ON operator.id = c.operator_user_id
`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT c.id, c.customer_user_id, customer.email,
       c.from_agent_id, from_user.email,
       c.to_agent_id, to_user.email,
       c.reason, COALESCE(operator.email,''), c.effective_at, c.created_at
FROM agent_customer_relation_changes c
JOIN users customer ON customer.id = c.customer_user_id AND customer.deleted_at IS NULL
LEFT JOIN agent_profiles from_agent ON from_agent.id = c.from_agent_id
LEFT JOIN users from_user ON from_user.id = from_agent.user_id
LEFT JOIN agent_profiles to_agent ON to_agent.id = c.to_agent_id
LEFT JOIN users to_user ON to_user.id = to_agent.user_id
LEFT JOIN users operator ON operator.id = c.operator_user_id
`+where+` ORDER BY c.created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []AgentCustomerRelationChange{}
	for rows.Next() {
		var item AgentCustomerRelationChange
		var fromID, toID sql.NullInt64
		var fromEmail, toEmail sql.NullString
		if err := rows.Scan(&item.ID, &item.CustomerUserID, &item.CustomerEmail, &fromID, &fromEmail, &toID, &toEmail, &item.Reason, &item.OperatorEmail, &item.EffectiveAt, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		if fromID.Valid {
			item.FromAgentID = &fromID.Int64
		}
		if fromEmail.Valid {
			item.FromAgentEmail = &fromEmail.String
		}
		if toID.Valid {
			item.ToAgentID = &toID.Int64
		}
		if toEmail.Valid {
			item.ToAgentEmail = &toEmail.String
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetCustomerRelation(ctx context.Context, id int64) (*AgentCustomer, error) {
	rows, err := r.db.QueryContext(ctx, customerSelectSQL()+` WHERE acr.id=$1 AND u.deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanCustomers(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	return &items[0], nil
}

func (r *Repository) ListCommissions(ctx context.Context, filter ListFilter) ([]AgentCommission, int64, error) {
	where, args := buildAgentIDWhere("acp.agent_id", filter.AgentID)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_commission_periods acp `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT acp.id, COALESCE(u.email,''), acp.order_id, acp.period_start_at, acp.period_end_at,
       acp.order_paid_amount, acp.confirmed_revenue, acp.rate_bps, acp.commission_amount,
       acp.reverse_amount, acp.reverse_reason_type, acp.status
FROM agent_commission_periods acp
LEFT JOIN users u ON u.id = acp.customer_user_id
`+where+` ORDER BY acp.period_end_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []AgentCommission{}
	for rows.Next() {
		var item AgentCommission
		var orderID sql.NullInt64
		var reverseReason sql.NullString
		if err := rows.Scan(&item.ID, &item.CustomerEmail, &orderID, &item.PeriodStartAt, &item.PeriodEndAt, &item.OrderPaidAmount, &item.ConfirmedRevenue, &item.RateBPS, &item.CommissionAmount, &item.ReverseAmount, &reverseReason, &item.Status); err != nil {
			return nil, 0, err
		}
		if orderID.Valid {
			item.OrderID = &orderID.Int64
		}
		if reverseReason.Valid {
			item.ReverseReasonType = &reverseReason.String
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListSettlements(ctx context.Context, filter ListFilter) ([]AgentSettlement, int64, error) {
	where, args := buildAgentIDWhere("s.agent_id", filter.AgentID)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM agent_settlements s
JOIN agent_profiles ap ON ap.id = s.agent_id
JOIN users u ON u.id = ap.user_id AND u.deleted_at IS NULL
`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, settlementSelectSQL()+where+` ORDER BY s.period_month DESC, s.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []AgentSettlement{}
	for rows.Next() {
		item, err := scanSettlementRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

type RegisterSettlementPaymentInput struct {
	Amount           int64
	PaymentMethod    *string
	PaymentReference *string
	PaidAt           *time.Time
	Remark           *string
	OperatorID       int64
}

func (r *Repository) RegisterSettlementPayment(ctx context.Context, id int64, in RegisterSettlementPaymentInput) (*AgentSettlement, error) {
	if in.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}
	method := optionalTrimmedString(in.PaymentMethod)
	if method != nil {
		switch *method {
		case "bank_transfer", "alipay", "wechat_pay", "cash", "other":
		default:
			return nil, fmt.Errorf("invalid payment method")
		}
	}
	reference := optionalTrimmedString(in.PaymentReference)
	remark := optionalTrimmedString(in.Remark)
	paidAt := time.Now().UTC()
	if in.PaidAt != nil {
		paidAt = *in.PaidAt
	}

	current, err := r.GetSettlementByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status != "payable" {
		return nil, fmt.Errorf("only payable settlement can register payment")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
INSERT INTO agent_settlement_payments (
    settlement_id, agent_id, amount, payment_method, payment_reference, paid_at, remark, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)`, id, current.AgentID, in.Amount, method, reference, paidAt, remark, in.OperatorID)
	if err != nil {
		if strings.Contains(err.Error(), "agent_settlement_payments_settlement_id") || strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("settlement payment already registered")
		}
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE agent_settlements
SET status='paid',
    paid_at=$1,
    paid_by_user_id=$2,
    payment_reference=COALESCE($3, payment_reference),
    updated_at=NOW()
WHERE id=$4`, paidAt, in.OperatorID, reference, id); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, in.OperatorID, "admin", "settlement.register_payment", "agent_settlement", id, current, map[string]any{
		"amount":            in.Amount,
		"payment_method":    method,
		"payment_reference": reference,
		"paid_at":           paidAt,
		"remark":            remark,
	}, remark); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetSettlementByID(ctx, id)
}

type AdjustSettlementInput struct {
	Amount           *int64
	ReverseAmount    *int64
	NetAmount        *int64
	Status           *string
	PaymentReference *string
	Reason           string
	OperatorID       int64
}

func (r *Repository) AdjustSettlement(ctx context.Context, id int64, in AdjustSettlementInput) (*AgentSettlement, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, fmt.Errorf("reason is required")
	}
	current, err := r.GetSettlementByID(ctx, id)
	if err != nil {
		return nil, err
	}

	amount := current.Amount
	reverseAmount := current.ReverseAmount
	netAmount := current.NetAmount
	status := current.Status
	if in.Amount != nil {
		amount = *in.Amount
	}
	if in.ReverseAmount != nil {
		reverseAmount = *in.ReverseAmount
	}
	if in.NetAmount != nil {
		netAmount = *in.NetAmount
	} else {
		netAmount = amount - reverseAmount
	}
	if in.Status != nil {
		status = strings.TrimSpace(*in.Status)
	}
	switch status {
	case "pending", "frozen", "payable", "paid", "reversed":
	default:
		return nil, fmt.Errorf("invalid settlement status")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
UPDATE agent_settlements
SET amount=$1,
    reverse_amount=$2,
    net_amount=$3,
    status=$4,
    min_amount_met=$3 >= 10000,
    payment_reference=COALESCE($5, payment_reference),
    updated_at=NOW()
WHERE id=$6`, amount, reverseAmount, netAmount, status, in.PaymentReference, id); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, in.OperatorID, "admin", "settlement.adjust", "agent_settlement", id, current, map[string]any{"amount": amount, "reverse_amount": reverseAmount, "net_amount": netAmount, "status": status, "payment_reference": in.PaymentReference}, &reason); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetSettlementByID(ctx, id)
}

func (r *Repository) GetSettlementByID(ctx context.Context, id int64) (*AgentSettlement, error) {
	rows, err := r.db.QueryContext(ctx, settlementSelectSQL()+` WHERE s.id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanSettlements(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	return &items[0], nil
}

func (r *Repository) ListChildren(ctx context.Context, parentAgentID int64, filter ListFilter) ([]AgentProfile, int64, error) {
	where, args := buildChildAgentWhere(parentAgentID, filter.Search)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_profiles ap JOIN users u ON u.id = ap.user_id `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, agentSelectSQL()+where+` ORDER BY ap.created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanAgents(rows)
	return items, total, err
}

func (r *Repository) ListInvites(ctx context.Context, inviterUserID int64, filter ListFilter) ([]AgentCustomer, int64, error) {
	where := ` WHERE ua.inviter_id = $1 AND u.deleted_at IS NULL`
	args := []any{inviterUserID}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		where += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.username ILIKE $%d)", len(args), len(args))
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_affiliates ua JOIN users u ON u.id = ua.user_id`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT 0::bigint, u.id, u.email, COALESCE(u.username,''), 'referral'::text,
       inviter_aff.aff_code, ap.id, COALESCE(inviter.email,''), sub_group.name, sub.expires_at,
       0::bigint, 'active'::text
FROM user_affiliates ua
JOIN users u ON u.id = ua.user_id AND u.deleted_at IS NULL
JOIN users inviter ON inviter.id = ua.inviter_id AND inviter.deleted_at IS NULL
JOIN agent_profiles ap ON ap.user_id = ua.inviter_id
LEFT JOIN user_affiliates inviter_aff ON inviter_aff.user_id = ua.inviter_id
LEFT JOIN LATERAL (
    SELECT us.expires_at, us.group_id
    FROM user_subscriptions us
    WHERE us.user_id = u.id AND us.deleted_at IS NULL
    ORDER BY us.expires_at DESC
    LIMIT 1
) sub ON true
LEFT JOIN groups sub_group ON sub_group.id = sub.group_id AND sub_group.deleted_at IS NULL
`+where+` ORDER BY ua.created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanCustomers(rows)
	return items, total, err
}

func (r *Repository) CreateChildAgent(ctx context.Context, parent AgentProfile, userID int64, rateBPS *int, operatorID int64) (*AgentProfile, error) {
	if parent.Level >= 3 {
		return nil, ErrForbidden
	}
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM user_affiliates WHERE user_id = $1 AND inviter_id = $2)`, userID, parent.UserID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("only referred users can be promoted to child agent")
	}
	input := CreateAgentInput{
		UserID:        userID,
		Level:         parent.Level + 1,
		ParentAgentID: &parent.ID,
		RateBPS:       rateBPS,
		OperatorID:    operatorID,
	}
	return r.CreateAgent(ctx, input)
}

func (r *Repository) UpdateChildRate(ctx context.Context, parent AgentProfile, childAgentID int64, rateBPS int, operatorID int64) (*AgentProfile, error) {
	child, err := r.GetAgentByID(ctx, childAgentID)
	if err != nil {
		return nil, err
	}
	if child.ParentAgentID == nil || *child.ParentAgentID != parent.ID {
		return nil, ErrForbidden
	}
	return r.UpdateAgent(ctx, childAgentID, UpdateAgentInput{RateBPS: &rateBPS, OperatorID: operatorID})
}

func (r *Repository) ListOrders(ctx context.Context, agentID int64, filter ListFilter) ([]AgentOrder, int64, error) {
	where := ` WHERE acr.status = 'active' AND acr.agent_id = $1 AND u.deleted_at IS NULL`
	args := []any{agentID}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		where += fmt.Sprintf(" AND (u.email ILIKE $%d OR po.out_trade_no ILIKE $%d)", len(args), len(args))
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM agent_customer_relations acr
JOIN users u ON u.id = acr.customer_user_id
JOIN payment_orders po ON po.user_id = acr.customer_user_id
`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT po.id, COALESCE(po.out_trade_no,''), u.email, COALESCE(po.status,''), ROUND(COALESCE(po.pay_amount, po.amount, 0) * 100)::bigint,
       po.paid_at, po.completed_at
FROM agent_customer_relations acr
JOIN users u ON u.id = acr.customer_user_id
JOIN payment_orders po ON po.user_id = acr.customer_user_id
`+where+` ORDER BY COALESCE(po.completed_at, po.paid_at, po.created_at) DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []AgentOrder{}
	for rows.Next() {
		var item AgentOrder
		var paidAt, completedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.OrderNo, &item.CustomerEmail, &item.Status, &item.PayAmount, &paidAt, &completedAt); err != nil {
			return nil, 0, err
		}
		if paidAt.Valid {
			item.PaidAt = &paidAt.Time
		}
		if completedAt.Valid {
			item.CompletedAt = &completedAt.Time
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) IsAgentAdminUser(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM agent_admin_users
    WHERE user_id = $1 AND status = 'active'
)`, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) ListAgentAdminUsers(ctx context.Context, filter ListFilter) ([]AgentAdminUser, int64, error) {
	where := ` WHERE 1=1`
	args := []any{}
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (email ILIKE $%d OR username ILIKE $%d)", len(args), len(args))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
WITH admin_users AS (
    SELECT 0::bigint AS id,
           u.id AS user_id,
           u.email,
           COALESCE(u.username, '') AS username,
           'base' AS source,
           0 AS source_order,
           u.status,
           NULL::text AS created_by_email,
           u.created_at,
           NULL::timestamptz AS revoked_at
    FROM users u
    WHERE u.deleted_at IS NULL AND u.role = 'admin'
    UNION ALL
    SELECT aau.id,
           u.id AS user_id,
           u.email,
           COALESCE(u.username, '') AS username,
           'delegated' AS source,
           1 AS source_order,
           aau.status,
           creator.email AS created_by_email,
           aau.created_at,
           aau.revoked_at
    FROM agent_admin_users aau
    JOIN users u ON u.id = aau.user_id AND u.deleted_at IS NULL
    LEFT JOIN users creator ON creator.id = aau.created_by
)
SELECT COUNT(*)
FROM admin_users
`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
WITH admin_users AS (
SELECT 0::bigint AS id,
       u.id AS user_id,
       u.email,
       COALESCE(u.username, '') AS username,
       'base' AS source,
       0 AS source_order,
       u.status,
       NULL::text AS created_by_email,
       u.created_at,
       NULL::timestamptz AS revoked_at
FROM users u
WHERE u.deleted_at IS NULL AND u.role = 'admin'
UNION ALL
SELECT aau.id,
       u.id AS user_id,
       u.email,
       COALESCE(u.username, '') AS username,
       'delegated' AS source,
       1 AS source_order,
       aau.status,
       creator.email AS created_by_email,
       aau.created_at,
       aau.revoked_at
FROM agent_admin_users aau
JOIN users u ON u.id = aau.user_id AND u.deleted_at IS NULL
LEFT JOIN users creator ON creator.id = aau.created_by
)
SELECT id, user_id, email, username, source, status, created_by_email, created_at, revoked_at
FROM admin_users
`+where+` ORDER BY source_order, created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanAgentAdminUsers(rows)
	return items, total, err
}

func (r *Repository) GrantAgentAdmin(ctx context.Context, userID, operatorID int64) (*AgentAdminUser, error) {
	user, err := r.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Role == "admin" {
		return nil, fmt.Errorf("base admin already has agent-admin permission")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var id int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO agent_admin_users (user_id, status, created_by, revoked_by, revoked_at, updated_at)
VALUES ($1, 'active', $2, NULL, NULL, NOW())
ON CONFLICT (user_id) DO UPDATE
SET status='active',
    created_by=EXCLUDED.created_by,
    revoked_by=NULL,
    revoked_at=NULL,
    updated_at=NOW()
RETURNING id`, userID, operatorID).Scan(&id); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, operatorID, "admin", "admin_user.grant", "agent_admin_user", id, nil, map[string]any{"user_id": userID, "email": user.Email}, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAgentAdminUserByID(ctx, id)
}

func (r *Repository) RevokeAgentAdmin(ctx context.Context, id, operatorID int64) (*AgentAdminUser, error) {
	current, err := r.GetAgentAdminUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
UPDATE agent_admin_users
SET status='disabled',
    revoked_by=$1,
    revoked_at=NOW(),
    updated_at=NOW()
WHERE id=$2`, operatorID, id); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, operatorID, "admin", "admin_user.revoke", "agent_admin_user", id, current, map[string]any{"status": "disabled"}, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAgentAdminUserByID(ctx, id)
}

func (r *Repository) GetAgentAdminUserByID(ctx context.Context, id int64) (*AgentAdminUser, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT aau.id,
       u.id AS user_id,
       u.email,
       COALESCE(u.username, '') AS username,
       'delegated' AS source,
       aau.status,
       creator.email AS created_by_email,
       aau.created_at,
       aau.revoked_at
FROM agent_admin_users aau
JOIN users u ON u.id = aau.user_id AND u.deleted_at IS NULL
LEFT JOIN users creator ON creator.id = aau.created_by
WHERE aau.id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanAgentAdminUsers(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	return &items[0], nil
}

func (r *Repository) ListAuditLogs(ctx context.Context, filter ListFilter) ([]AgentAuditLog, int64, error) {
	where := ""
	args := []any{}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		where = fmt.Sprintf(" WHERE l.action ILIKE $%d OR l.target_type ILIKE $%d OR COALESCE(u.email,'') ILIKE $%d", len(args), len(args), len(args))
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_audit_logs l LEFT JOIN users u ON u.id=l.operator_user_id`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit(), filter.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT l.id, COALESCE(u.email,''), l.operator_role, l.action, l.target_type, l.target_id, l.reason, l.created_at
FROM agent_audit_logs l
LEFT JOIN users u ON u.id = l.operator_user_id
`+where+` ORDER BY l.created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []AgentAuditLog{}
	for rows.Next() {
		var item AgentAuditLog
		var reason sql.NullString
		if err := rows.Scan(&item.ID, &item.OperatorEmail, &item.OperatorRole, &item.Action, &item.TargetType, &item.TargetID, &reason, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		if reason.Valid {
			item.Reason = &reason.String
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetSettings(ctx context.Context) (AgentAdminSettings, error) {
	values, updatedAt, err := r.getSettingsMap(ctx)
	if err != nil {
		return AgentAdminSettings{}, err
	}
	return AgentAdminSettings{
		TurnstileEnabled: parseBoolSetting(values["turnstile_enabled"]),
		TurnstileSiteKey: strings.TrimSpace(values["turnstile_site_key"]),
		UpdatedAt:        updatedAt,
	}, nil
}

func (r *Repository) GetPublicSettings(ctx context.Context) (PublicSettings, error) {
	settings, err := r.GetSettings(ctx)
	if err != nil {
		return PublicSettings{}, err
	}
	return PublicSettings{
		TurnstileEnabled: settings.TurnstileEnabled,
		TurnstileSiteKey: settings.TurnstileSiteKey,
	}, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, in AgentAdminSettings, operatorID int64) (AgentAdminSettings, error) {
	siteKey := strings.TrimSpace(in.TurnstileSiteKey)
	if in.TurnstileEnabled && siteKey == "" {
		return AgentAdminSettings{}, fmt.Errorf("turnstile site key is required when enabled")
	}

	current, err := r.GetSettings(ctx)
	if err != nil {
		return AgentAdminSettings{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return AgentAdminSettings{}, err
	}
	defer tx.Rollback()

	pairs := map[string]string{
		"turnstile_enabled":  formatBoolSetting(in.TurnstileEnabled),
		"turnstile_site_key": siteKey,
	}
	for key, value := range pairs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO agent_admin_settings (key, value, updated_by, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()`, key, value, operatorID); err != nil {
			return AgentAdminSettings{}, err
		}
	}
	next := AgentAdminSettings{TurnstileEnabled: in.TurnstileEnabled, TurnstileSiteKey: siteKey}
	if err := insertAudit(ctx, tx, operatorID, "admin", "settings.update", "agent_admin_settings", 0, current, next, nil); err != nil {
		return AgentAdminSettings{}, err
	}
	if err := tx.Commit(); err != nil {
		return AgentAdminSettings{}, err
	}
	return r.GetSettings(ctx)
}

func (r *Repository) getSettingsMap(ctx context.Context) (map[string]string, time.Time, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value, updated_at FROM agent_admin_settings`)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer rows.Close()

	values := map[string]string{
		"turnstile_enabled":  "false",
		"turnstile_site_key": "",
	}
	var latest time.Time
	for rows.Next() {
		var key, value string
		var updatedAt time.Time
		if err := rows.Scan(&key, &value, &updatedAt); err != nil {
			return nil, time.Time{}, err
		}
		values[key] = value
		if updatedAt.After(latest) {
			latest = updatedAt
		}
	}
	if latest.IsZero() {
		latest = time.Now().UTC()
	}
	return values, latest, rows.Err()
}

func (r *Repository) InsertAudit(ctx context.Context, operatorID int64, role, action, targetType string, targetID int64, before, after any) error {
	return insertAudit(ctx, r.db, operatorID, role, action, targetType, targetID, before, after, nil)
}

func (r *Repository) validateAgentInput(ctx context.Context, level int, parentAgentID *int64, rateBPS *int) error {
	if level < 1 || level > 3 {
		return fmt.Errorf("invalid agent level")
	}
	if level == 1 && parentAgentID != nil {
		return fmt.Errorf("level 1 agent must not have parent")
	}
	if level > 1 && parentAgentID == nil {
		return fmt.Errorf("level %d agent requires parent", level)
	}
	if rateBPS != nil && (*rateBPS < 0 || *rateBPS > 10000) {
		return fmt.Errorf("invalid commission rate")
	}
	if parentAgentID == nil {
		return nil
	}
	parent, err := r.GetAgentByID(ctx, *parentAgentID)
	if err != nil {
		return fmt.Errorf("parent agent not found")
	}
	if parent.Level != level-1 {
		return fmt.Errorf("parent agent level must be %d", level-1)
	}
	if rateBPS != nil && *rateBPS >= parent.RateBPS {
		return fmt.Errorf("child rate must be lower than parent rate")
	}
	return nil
}

func (r *Repository) isDescendantAgent(ctx context.Context, candidateID int64, ancestorID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
WITH RECURSIVE descendants AS (
    SELECT id
    FROM agent_profiles
    WHERE parent_agent_id = $1

    UNION ALL

    SELECT child.id
    FROM agent_profiles child
    JOIN descendants d ON child.parent_agent_id = d.id
)
SELECT EXISTS (SELECT 1 FROM descendants WHERE id = $2)`, ancestorID, candidateID).Scan(&exists)
	return exists, err
}

func optionalTrimmedString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func insertAudit(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, operatorID int64, role, action, targetType string, targetID int64, before, after any, reason *string) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if before == nil {
		beforeJSON = nil
	}
	if after == nil {
		afterJSON = nil
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO agent_audit_logs (operator_user_id, operator_role, action, target_type, target_id, before_data, after_data, reason)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, operatorID, role, action, targetType, targetID, nullableJSON(beforeJSON), nullableJSON(afterJSON), reason)
	return err
}

func nullableJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}

func parseBoolSetting(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func formatBoolSetting(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func defaultRateBPS(level int) int {
	switch level {
	case 1:
		return 2000
	case 2:
		return 1500
	case 3:
		return 1000
	default:
		return 0
	}
}

func buildAgentWhere(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return " WHERE u.deleted_at IS NULL", nil
	}
	return " WHERE u.deleted_at IS NULL AND (u.email ILIKE $1 OR u.username ILIKE $1)", []any{"%" + search + "%"}
}

func buildChildAgentWhere(parentAgentID int64, search string) (string, []any) {
	where := " WHERE ap.parent_agent_id = $1 AND u.deleted_at IS NULL"
	args := []any{parentAgentID}
	search = strings.TrimSpace(search)
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.username ILIKE $%d)", len(args), len(args))
	}
	return where, args
}

func buildAgentIDWhere(column string, agentID *int64) (string, []any) {
	if agentID == nil {
		return "", nil
	}
	return " WHERE " + column + " = $1", []any{*agentID}
}

func settlementSelectSQL() string {
	return `
SELECT s.id, s.agent_id, u.email, to_char(s.period_month, 'YYYY-MM'), s.amount, s.reverse_amount,
       s.net_amount, s.status, s.frozen_until, s.paid_at,
       p.amount, p.payment_method, COALESCE(p.payment_reference, s.payment_reference), p.remark,
       p.created_at, operator.email
FROM agent_settlements s
JOIN agent_profiles ap ON ap.id = s.agent_id
JOIN users u ON u.id = ap.user_id AND u.deleted_at IS NULL
LEFT JOIN agent_settlement_payments p ON p.settlement_id = s.id
LEFT JOIN users operator ON operator.id = p.created_by`
}

func scanSettlements(rows *sql.Rows) ([]AgentSettlement, error) {
	items := []AgentSettlement{}
	for rows.Next() {
		item, err := scanSettlementRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanAgentAdminUsers(rows *sql.Rows) ([]AgentAdminUser, error) {
	items := []AgentAdminUser{}
	for rows.Next() {
		var item AgentAdminUser
		var createdBy sql.NullString
		var revokedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.UserID, &item.Email, &item.Username, &item.Source, &item.Status, &createdBy, &item.CreatedAt, &revokedAt); err != nil {
			return nil, err
		}
		if createdBy.Valid {
			item.CreatedByEmail = &createdBy.String
		}
		if revokedAt.Valid {
			item.RevokedAt = &revokedAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type settlementScanner interface {
	Scan(dest ...any) error
}

func scanSettlementRow(row settlementScanner) (AgentSettlement, error) {
	var item AgentSettlement
	var frozenUntil, paidAt, paymentRegisteredAt sql.NullTime
	var paymentAmount sql.NullInt64
	var paymentMethod, paymentReference, paymentRemark, paymentOperatorEmail sql.NullString
	err := row.Scan(
		&item.ID, &item.AgentID, &item.AgentEmail, &item.PeriodMonth, &item.Amount, &item.ReverseAmount,
		&item.NetAmount, &item.Status, &frozenUntil, &paidAt, &paymentAmount, &paymentMethod,
		&paymentReference, &paymentRemark, &paymentRegisteredAt, &paymentOperatorEmail,
	)
	if err != nil {
		return item, err
	}
	if frozenUntil.Valid {
		item.FrozenUntil = &frozenUntil.Time
	}
	if paidAt.Valid {
		item.PaidAt = &paidAt.Time
	}
	if paymentAmount.Valid {
		item.PaymentAmount = &paymentAmount.Int64
	}
	if paymentMethod.Valid {
		item.PaymentMethod = &paymentMethod.String
	}
	if paymentReference.Valid {
		item.PaymentReference = &paymentReference.String
	}
	if paymentRemark.Valid {
		item.PaymentRemark = &paymentRemark.String
	}
	if paymentRegisteredAt.Valid {
		item.PaymentRegisteredAt = &paymentRegisteredAt.Time
	}
	if paymentOperatorEmail.Valid {
		item.PaymentOperatorEmail = &paymentOperatorEmail.String
	}
	return item, nil
}

func agentSelectSQL() string {
	return `
SELECT ap.id, ap.user_id, COALESCE(u.username,''), u.email, ap.level, ap.parent_agent_id,
       COALESCE(parent_user.username,''), parent_user.email, ap.status,
       COALESCE(rate.rate_bps, 0), COALESCE(children.children_count,0)::bigint,
       COALESCE(customers.customers_count,0)::bigint,
       COALESCE(payable.payable_amount,0)::bigint,
       COALESCE(frozen.frozen_amount,0)::bigint,
       ap.created_at, ap.disabled_at
FROM agent_profiles ap
JOIN users u ON u.id = ap.user_id
LEFT JOIN agent_profiles parent ON parent.id = ap.parent_agent_id
LEFT JOIN users parent_user ON parent_user.id = parent.user_id AND parent_user.deleted_at IS NULL
LEFT JOIN agent_commission_rates rate ON rate.agent_id = ap.id AND rate.expired_at IS NULL
LEFT JOIN (
    SELECT child.parent_agent_id, COUNT(*) AS children_count
    FROM agent_profiles child
    JOIN users child_user ON child_user.id = child.user_id AND child_user.deleted_at IS NULL
    WHERE child.parent_agent_id IS NOT NULL
    GROUP BY child.parent_agent_id
) children ON children.parent_agent_id = ap.id
LEFT JOIN (
    SELECT acr.agent_id, COUNT(*) AS customers_count
    FROM agent_customer_relations acr
    JOIN users customer ON customer.id = acr.customer_user_id AND customer.deleted_at IS NULL
    WHERE acr.status='active'
    GROUP BY acr.agent_id
) customers ON customers.agent_id = ap.id
LEFT JOIN (
    SELECT agent_id, SUM(net_amount) AS payable_amount FROM agent_settlements WHERE status IN ('payable','pending') GROUP BY agent_id
) payable ON payable.agent_id = ap.id
LEFT JOIN (
    SELECT agent_id, SUM(commission_amount) AS frozen_amount FROM agent_commission_periods WHERE status='frozen' GROUP BY agent_id
) frozen ON frozen.agent_id = ap.id`
}

func scanOneAgent(rows *sql.Rows) (*AgentProfile, error) {
	items, err := scanAgents(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	return &items[0], nil
}

func scanAgents(rows *sql.Rows) ([]AgentProfile, error) {
	items := []AgentProfile{}
	for rows.Next() {
		var item AgentProfile
		var parentID sql.NullInt64
		var parentUsername, parentEmail sql.NullString
		var disabledAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.UserID, &item.Username, &item.Email, &item.Level, &parentID, &parentUsername, &parentEmail, &item.Status, &item.RateBPS, &item.ChildrenCount, &item.CustomersCount, &item.PayableAmount, &item.FrozenAmount, &item.CreatedAt, &disabledAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			item.ParentAgentID = &parentID.Int64
		}
		if parentUsername.Valid {
			item.ParentUsername = &parentUsername.String
		}
		if parentEmail.Valid {
			item.ParentEmail = &parentEmail.String
		}
		if disabledAt.Valid {
			item.DisabledAt = &disabledAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func customerSelectSQL() string {
	return `
SELECT acr.id, u.id, u.email, COALESCE(u.username,''), acr.source, acr.source_referral_code,
       acr.agent_id, agent_user.email, sub_group.name, sub.expires_at,
       COALESCE(revenue.confirmed_revenue,0)::bigint, acr.status
FROM agent_customer_relations acr
JOIN users u ON u.id = acr.customer_user_id
JOIN agent_profiles ap ON ap.id = acr.agent_id
JOIN users agent_user ON agent_user.id = ap.user_id AND agent_user.deleted_at IS NULL
LEFT JOIN LATERAL (
    SELECT us.expires_at, us.group_id
    FROM user_subscriptions us
    WHERE us.user_id = u.id AND us.deleted_at IS NULL
    ORDER BY us.expires_at DESC
    LIMIT 1
) sub ON true
LEFT JOIN groups sub_group ON sub_group.id = sub.group_id AND sub_group.deleted_at IS NULL
LEFT JOIN (
    SELECT customer_user_id, agent_id, SUM(confirmed_revenue) AS confirmed_revenue
    FROM agent_commission_periods
    GROUP BY customer_user_id, agent_id
) revenue ON revenue.customer_user_id = acr.customer_user_id AND revenue.agent_id = acr.agent_id`
}

func scanCustomers(rows *sql.Rows) ([]AgentCustomer, error) {
	items := []AgentCustomer{}
	for rows.Next() {
		var item AgentCustomer
		var code, subName sql.NullString
		var periodEnd sql.NullTime
		if err := rows.Scan(&item.ID, &item.UserID, &item.Email, &item.Username, &item.Source, &code, &item.AgentID, &item.AgentName, &subName, &periodEnd, &item.ConfirmedRevenue, &item.Status); err != nil {
			return nil, err
		}
		if code.Valid {
			item.SourceReferralCode = &code.String
		}
		if subName.Valid {
			item.SubscriptionName = &subName.String
		}
		if periodEnd.Valid {
			item.PeriodEndAt = &periodEnd.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

var ErrForbidden = errors.New("forbidden")
