package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type agentRepository struct {
	client *dbent.Client
}

func NewAgentRepository(client *dbent.Client, _ *sql.DB) service.AgentRepository {
	return &agentRepository{client: client}
}

func (r *agentRepository) GetAgent(ctx context.Context, id int64) (*service.AgentProfile, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, agentProfileSelectSQL()+` WHERE ap.id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("query agent: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrAgentNotFound
	}
	profile, err := scanAgentProfile(rows)
	if err != nil {
		return nil, err
	}
	return profile, rows.Close()
}

func (r *agentRepository) GetAgentByUserID(ctx context.Context, userID int64) (*service.AgentProfile, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, agentProfileSelectSQL()+` WHERE ap.user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("query agent by user: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrAgentNotFound
	}
	profile, err := scanAgentProfile(rows)
	if err != nil {
		return nil, err
	}
	return profile, rows.Close()
}

func (r *agentRepository) ListAgents(ctx context.Context, filter service.AgentListFilter) ([]service.AgentProfile, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildAgentSearchWhere(filter.Search)
	countSQL := "SELECT COUNT(*) FROM agent_profiles ap JOIN users u ON u.id = ap.user_id" + where
	var total int64
	if err := queryOne(ctx, client, countSQL, args, &total); err != nil {
		return nil, 0, fmt.Errorf("count agents: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, agentProfileSelectSQL()+where+` ORDER BY ap.created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list agents: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items, err := scanAgentProfiles(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, rows.Close()
}

func (r *agentRepository) CreateAgent(ctx context.Context, input service.CreateAgentInput, rateBPS int) (*service.AgentProfile, error) {
	var id int64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := queryOne(txCtx, txClient, `
INSERT INTO agent_profiles (user_id, level, parent_agent_id, status, created_by, created_at, updated_at)
VALUES ($1, $2, $3, 'active', $4, NOW(), NOW())
RETURNING id`, []any{input.UserID, input.Level, nullableInt64Value(input.ParentAgentID), nullablePositiveInt64(input.OperatorID)}, &id); err != nil {
			if isUniqueConstraintViolation(err) {
				return service.ErrAgentAlreadyExists
			}
			return fmt.Errorf("insert agent profile: %w", err)
		}
		if _, err := txClient.ExecContext(txCtx, `
INSERT INTO agent_commission_rates (agent_id, rate_bps, set_by_user_id, effective_at, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW(), NOW())`, id, rateBPS, nullablePositiveInt64(input.OperatorID)); err != nil {
			return fmt.Errorf("insert agent rate: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetAgent(ctx, id)
}

func (r *agentRepository) UpdateAgent(ctx context.Context, id int64, input service.UpdateAgentInput, rateBPS *int) (*service.AgentProfile, error) {
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		args := []any{id}
		levelExpr := "level"
		if input.Level != nil {
			args = append(args, *input.Level)
			levelExpr = fmt.Sprintf("$%d", len(args))
		}
		parentExpr := "parent_agent_id"
		if input.ParentAgentID != nil || (input.Level != nil && *input.Level == 1) {
			args = append(args, nullableInt64Value(input.ParentAgentID))
			parentExpr = fmt.Sprintf("$%d", len(args))
		}
		query := fmt.Sprintf(`
UPDATE agent_profiles
SET level = %s,
    parent_agent_id = %s,
    updated_at = NOW()
WHERE id = $1`, levelExpr, parentExpr)
		res, err := txClient.ExecContext(txCtx, query, args...)
		if err != nil {
			return fmt.Errorf("update agent profile: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return service.ErrAgentNotFound
		}
		if rateBPS != nil {
			if _, err := txClient.ExecContext(txCtx, `
UPDATE agent_commission_rates
SET expired_at = NOW(), updated_at = NOW()
WHERE agent_id = $1 AND expired_at IS NULL`, id); err != nil {
				return fmt.Errorf("expire agent rate: %w", err)
			}
			if _, err := txClient.ExecContext(txCtx, `
INSERT INTO agent_commission_rates (agent_id, rate_bps, set_by_user_id, effective_at, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW(), NOW())`, id, *rateBPS, nullablePositiveInt64(input.OperatorID)); err != nil {
				return fmt.Errorf("insert agent rate: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetAgent(ctx, id)
}

func (r *agentRepository) SetAgentStatus(ctx context.Context, id int64, status string, operatorID int64) (*service.AgentProfile, error) {
	client := clientFromContext(ctx, r.client)
	var disabledExpr string
	if status == service.AgentStatusDisabled {
		disabledExpr = "NOW()"
	} else {
		disabledExpr = "NULL"
	}
	res, err := client.ExecContext(ctx, fmt.Sprintf(`
UPDATE agent_profiles
SET status = $1,
    disabled_at = %s,
    updated_at = NOW()
WHERE id = $2`, disabledExpr), status, id)
	if err != nil {
		return nil, fmt.Errorf("set agent status: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, service.ErrAgentNotFound
	}
	return r.GetAgent(ctx, id)
}

func (r *agentRepository) AssignCustomer(ctx context.Context, input service.AssignAgentCustomerInput) (*service.AgentCustomer, error) {
	var relationID int64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		var previous sql.NullInt64
		if err := queryOne(txCtx, txClient, `
SELECT agent_id
FROM agent_customer_relations
WHERE customer_user_id = $1 AND status = 'active' AND expired_at IS NULL
FOR UPDATE`, []any{input.CustomerUserID}, &previous); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("lock existing customer relation: %w", err)
		}
		if _, err := txClient.ExecContext(txCtx, `
UPDATE agent_customer_relations
SET status = 'expired', expired_at = NOW(), updated_at = NOW()
WHERE customer_user_id = $1 AND status = 'active' AND expired_at IS NULL`, input.CustomerUserID); err != nil {
			return fmt.Errorf("expire customer relation: %w", err)
		}
		if err := queryOne(txCtx, txClient, `
INSERT INTO agent_customer_relations (agent_id, customer_user_id, source, effective_at, status, created_by, created_at, updated_at)
VALUES ($1, $2, 'manual', NOW(), 'active', $3, NOW(), NOW())
RETURNING id`, []any{input.AgentID, input.CustomerUserID, nullablePositiveInt64(input.OperatorID)}, &relationID); err != nil {
			return fmt.Errorf("insert customer relation: %w", err)
		}
		var previousArg any
		if previous.Valid {
			previousArg = previous.Int64
		}
		if _, err := txClient.ExecContext(txCtx, `
INSERT INTO agent_customer_relation_changes (customer_user_id, from_agent_id, to_agent_id, reason, operator_user_id, effective_at, created_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`, input.CustomerUserID, previousArg, input.AgentID, input.Reason, nullablePositiveInt64(input.OperatorID)); err != nil {
			return fmt.Errorf("insert customer relation change: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.getCustomerRelation(ctx, relationID)
}

func (r *agentRepository) ListCustomers(ctx context.Context, filter service.AgentListFilter) ([]service.AgentCustomer, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildCustomerWhere(filter)
	var total int64
	if err := queryOne(ctx, client, `
SELECT COUNT(*)
FROM agent_customer_relations acr
JOIN users cu ON cu.id = acr.customer_user_id
JOIN agent_profiles ap ON ap.id = acr.agent_id
JOIN users au ON au.id = ap.user_id`+where, args, &total); err != nil {
		return nil, 0, fmt.Errorf("count agent customers: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, agentCustomerSelectSQL()+where+` ORDER BY acr.created_at DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent customers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items, err := scanAgentCustomers(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, rows.Close()
}

func (r *agentRepository) ListCommissions(ctx context.Context, filter service.AgentListFilter) ([]service.AgentCommission, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildCommissionWhere(filter)
	var total int64
	if err := queryOne(ctx, client, `
SELECT COUNT(*)
FROM agent_commission_periods acp
JOIN users cu ON cu.id = acp.customer_user_id
JOIN agent_profiles ap ON ap.id = acp.agent_id
JOIN users au ON au.id = ap.user_id`+where, args, &total); err != nil {
		return nil, 0, fmt.Errorf("count agent commissions: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, `
SELECT acp.id,
       COALESCE(cu.email, ''),
       acp.order_id,
       acp.period_start_at,
       acp.period_end_at,
       acp.order_paid_amount,
       acp.confirmed_revenue,
       acp.rate_bps,
       acp.commission_amount,
       acp.reverse_amount,
       acp.reverse_reason_type,
       acp.status,
       acp.frozen_until
FROM agent_commission_periods acp
JOIN users cu ON cu.id = acp.customer_user_id
JOIN agent_profiles ap ON ap.id = acp.agent_id
JOIN users au ON au.id = ap.user_id`+where+` ORDER BY acp.period_end_at DESC, acp.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent commissions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items, err := scanAgentCommissions(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, rows.Close()
}

func (r *agentRepository) ListSettlements(ctx context.Context, filter service.AgentListFilter) ([]service.AgentSettlement, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildSettlementWhere(filter)
	var total int64
	if err := queryOne(ctx, client, `
SELECT COUNT(*)
FROM agent_settlements s
JOIN agent_profiles ap ON ap.id = s.agent_id
JOIN users au ON au.id = ap.user_id`+where, args, &total); err != nil {
		return nil, 0, fmt.Errorf("count agent settlements: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, `
SELECT s.id,
       s.agent_id,
       COALESCE(au.email, ''),
       to_char(s.period_month, 'YYYY-MM'),
       s.amount,
       s.reverse_amount,
       s.net_amount,
       s.status,
       s.frozen_until,
       s.paid_at,
       s.payment_reference
FROM agent_settlements s
JOIN agent_profiles ap ON ap.id = s.agent_id
JOIN users au ON au.id = ap.user_id`+where+` ORDER BY s.period_month DESC, s.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent settlements: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items, err := scanAgentSettlements(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, rows.Close()
}

func (r *agentRepository) MarkSettlementPaid(ctx context.Context, id, operatorID int64) (*service.AgentSettlement, error) {
	client := clientFromContext(ctx, r.client)
	res, err := client.ExecContext(ctx, `
UPDATE agent_settlements
SET status = 'paid',
    paid_at = NOW(),
    paid_by_user_id = $1,
    updated_at = NOW()
WHERE id = $2 AND status IN ('payable', 'pending', 'frozen')`, nullablePositiveInt64(operatorID), id)
	if err != nil {
		return nil, fmt.Errorf("mark settlement paid: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, service.ErrAgentNotFound
	}
	return r.getSettlement(ctx, id)
}

func (r *agentRepository) ListAuditLogs(ctx context.Context, filter service.AgentListFilter) ([]service.AgentAuditLog, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildAuditWhere(filter.Search)
	var total int64
	if err := queryOne(ctx, client, `
SELECT COUNT(*)
FROM agent_audit_logs l
LEFT JOIN users u ON u.id = l.operator_user_id`+where, args, &total); err != nil {
		return nil, 0, fmt.Errorf("count agent audit logs: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, `
SELECT l.id,
       l.operator_user_id,
       COALESCE(u.email, ''),
       l.operator_role,
       l.action,
       l.target_type,
       l.target_id,
       l.reason,
       l.created_at
FROM agent_audit_logs l
LEFT JOIN users u ON u.id = l.operator_user_id`+where+` ORDER BY l.created_at DESC, l.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent audit logs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items, err := scanAgentAuditLogs(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, rows.Close()
}

func (r *agentRepository) CreateAuditLog(ctx context.Context, input service.AgentAuditInput) error {
	client := clientFromContext(ctx, r.client)
	beforeJSON, err := nullableJSON(input.BeforeData)
	if err != nil {
		return err
	}
	afterJSON, err := nullableJSON(input.AfterData)
	if err != nil {
		return err
	}
	_, err = client.ExecContext(ctx, `
INSERT INTO agent_audit_logs (operator_user_id, operator_role, action, target_type, target_id, before_data, after_data, reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		nullableInt64Value(input.OperatorUserID), input.OperatorRole, input.Action, input.TargetType, input.TargetID, beforeJSON, afterJSON, nullableStringValue(input.Reason))
	if err != nil {
		return fmt.Errorf("insert agent audit log: %w", err)
	}
	return nil
}

func (r *agentRepository) GetSummary(ctx context.Context, agentID *int64) (*service.AgentSummary, error) {
	client := clientFromContext(ctx, r.client)
	var summary service.AgentSummary
	if agentID == nil {
		if err := queryOne(ctx, client, `
SELECT COUNT(*)::bigint,
       COUNT(*) FILTER (WHERE status = 'active')::bigint,
       COUNT(*) FILTER (WHERE status = 'disabled')::bigint
FROM agent_profiles`, nil, &summary.TotalAgents, &summary.ActiveAgents, &summary.DisabledAgents); err != nil {
			return nil, fmt.Errorf("query agent summary counts: %w", err)
		}
	} else {
		if err := queryOne(ctx, client, `
SELECT COUNT(*)::bigint,
       COUNT(*) FILTER (WHERE status = 'active')::bigint,
       COUNT(*) FILTER (WHERE status = 'disabled')::bigint
FROM agent_profiles
WHERE parent_agent_id = $1`, []any{*agentID}, &summary.TotalAgents, &summary.ActiveAgents, &summary.DisabledAgents); err != nil {
			return nil, fmt.Errorf("query child agent summary counts: %w", err)
		}
		summary.ChildAgents = summary.TotalAgents
	}
	customerWhere := ""
	args := []any{}
	if agentID != nil {
		customerWhere = " AND agent_id = $1"
		args = append(args, *agentID)
	}
	if err := queryOne(ctx, client, `
SELECT COUNT(*)::bigint
FROM agent_customer_relations
WHERE status = 'active' AND expired_at IS NULL`+customerWhere, args, &summary.DirectCustomers); err != nil {
		return nil, fmt.Errorf("query direct customer summary: %w", err)
	}
	commissionWhere := ""
	if agentID != nil {
		commissionWhere = " WHERE agent_id = $1"
	}
	if err := queryOne(ctx, client, `
SELECT COALESCE(SUM(confirmed_revenue), 0)::bigint,
       COALESCE(SUM(commission_amount), 0)::bigint,
       COALESCE(SUM(reverse_amount), 0)::bigint
FROM agent_commission_periods`+commissionWhere, args, &summary.ConfirmedRevenue, &summary.CommissionAmount, &summary.ReversedAmount); err != nil {
		return nil, fmt.Errorf("query commission summary: %w", err)
	}
	settlementWhere := "WHERE status IN ('payable', 'pending', 'frozen')"
	if agentID != nil {
		settlementWhere += " AND agent_id = $1"
	}
	if err := queryOne(ctx, client, `
SELECT COALESCE(SUM(net_amount), 0)::bigint
FROM agent_settlements `+settlementWhere, args, &summary.PayableAmount); err != nil {
		return nil, fmt.Errorf("query settlement summary: %w", err)
	}
	if agentID == nil {
		if err := queryOne(ctx, client, `SELECT COUNT(*)::bigint FROM agent_profiles WHERE parent_agent_id IS NOT NULL`, nil, &summary.ChildAgents); err != nil {
			return nil, fmt.Errorf("query child agent count: %w", err)
		}
	}
	return &summary, nil
}

func (r *agentRepository) getCustomerRelation(ctx context.Context, id int64) (*service.AgentCustomer, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, agentCustomerSelectSQL()+` WHERE acr.id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrAgentNotFound
	}
	customer, err := scanAgentCustomer(rows)
	if err != nil {
		return nil, err
	}
	return customer, rows.Close()
}

func (r *agentRepository) getSettlement(ctx context.Context, id int64) (*service.AgentSettlement, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
SELECT s.id,
       s.agent_id,
       COALESCE(au.email, ''),
       to_char(s.period_month, 'YYYY-MM'),
       s.amount,
       s.reverse_amount,
       s.net_amount,
       s.status,
       s.frozen_until,
       s.paid_at,
       s.payment_reference
FROM agent_settlements s
JOIN agent_profiles ap ON ap.id = s.agent_id
JOIN users au ON au.id = ap.user_id
WHERE s.id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrAgentNotFound
	}
	settlement, err := scanAgentSettlement(rows)
	if err != nil {
		return nil, err
	}
	return settlement, rows.Close()
}

func (r *agentRepository) withTx(ctx context.Context, fn func(txCtx context.Context, txClient *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin agent transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit agent transaction: %w", err)
	}
	return nil
}

func agentProfileSelectSQL() string {
	return `
SELECT ap.id,
       ap.user_id,
       COALESCE(u.username, ''),
       COALESCE(u.email, ''),
       ap.level,
       ap.parent_agent_id,
       COALESCE(pu.username, ''),
       COALESCE(pu.email, ''),
       ap.status,
       COALESCE(rate.rate_bps, 0),
       COALESCE(children.children_count, 0),
       COALESCE(customers.customers_count, 0),
       COALESCE(payable.payable_amount, 0),
       COALESCE(frozen.frozen_amount, 0),
       ap.created_at,
       ap.disabled_at
FROM agent_profiles ap
JOIN users u ON u.id = ap.user_id
LEFT JOIN agent_profiles pap ON pap.id = ap.parent_agent_id
LEFT JOIN users pu ON pu.id = pap.user_id
LEFT JOIN agent_commission_rates rate ON rate.agent_id = ap.id AND rate.expired_at IS NULL
LEFT JOIN (
    SELECT parent_agent_id, COUNT(*)::integer AS children_count
    FROM agent_profiles
    WHERE parent_agent_id IS NOT NULL
    GROUP BY parent_agent_id
) children ON children.parent_agent_id = ap.id
LEFT JOIN (
    SELECT agent_id, COUNT(*)::integer AS customers_count
    FROM agent_customer_relations
    WHERE status = 'active' AND expired_at IS NULL
    GROUP BY agent_id
) customers ON customers.agent_id = ap.id
LEFT JOIN (
    SELECT agent_id, SUM(net_amount)::bigint AS payable_amount
    FROM agent_settlements
    WHERE status IN ('payable', 'pending')
    GROUP BY agent_id
) payable ON payable.agent_id = ap.id
LEFT JOIN (
    SELECT agent_id, SUM(net_amount)::bigint AS frozen_amount
    FROM agent_settlements
    WHERE status = 'frozen'
    GROUP BY agent_id
) frozen ON frozen.agent_id = ap.id`
}

func agentCustomerSelectSQL() string {
	return `
SELECT acr.id,
       acr.customer_user_id,
       COALESCE(cu.email, ''),
       COALESCE(cu.username, ''),
       acr.source,
       acr.source_referral_code,
       acr.agent_id,
       COALESCE(NULLIF(au.username, ''), au.email, ''),
       NULL::text AS subscription_name,
       NULL::timestamptz AS period_end_at,
       COALESCE(revenue.confirmed_revenue, 0),
       acr.status
FROM agent_customer_relations acr
JOIN users cu ON cu.id = acr.customer_user_id
JOIN agent_profiles ap ON ap.id = acr.agent_id
JOIN users au ON au.id = ap.user_id
LEFT JOIN (
    SELECT customer_user_id, agent_id, SUM(confirmed_revenue)::bigint AS confirmed_revenue
    FROM agent_commission_periods
    GROUP BY customer_user_id, agent_id
) revenue ON revenue.customer_user_id = acr.customer_user_id AND revenue.agent_id = acr.agent_id`
}

func buildAgentSearchWhere(search string) (string, []any) {
	if search == "" {
		return "", nil
	}
	return " WHERE (u.email ILIKE $1 OR u.username ILIKE $1)", []any{"%" + search + "%"}
}

func buildCustomerWhere(filter service.AgentListFilter) (string, []any) {
	conditions := []string{"acr.status = 'active'", "acr.expired_at IS NULL"}
	args := []any{}
	if filter.AgentID != nil {
		args = append(args, *filter.AgentID)
		conditions = append(conditions, fmt.Sprintf("acr.agent_id = $%d", len(args)))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf("(cu.email ILIKE $%d OR cu.username ILIKE $%d OR au.email ILIKE $%d OR au.username ILIKE $%d)", len(args), len(args), len(args), len(args)))
	}
	return " WHERE " + joinConditions(conditions), args
}

func buildCommissionWhere(filter service.AgentListFilter) (string, []any) {
	conditions := []string{}
	args := []any{}
	if filter.AgentID != nil {
		args = append(args, *filter.AgentID)
		conditions = append(conditions, fmt.Sprintf("acp.agent_id = $%d", len(args)))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf("(cu.email ILIKE $%d OR au.email ILIKE $%d)", len(args), len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + joinConditions(conditions), args
}

func buildSettlementWhere(filter service.AgentListFilter) (string, []any) {
	conditions := []string{}
	args := []any{}
	if filter.AgentID != nil {
		args = append(args, *filter.AgentID)
		conditions = append(conditions, fmt.Sprintf("s.agent_id = $%d", len(args)))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf("(au.email ILIKE $%d OR au.username ILIKE $%d)", len(args), len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + joinConditions(conditions), args
}

func buildAuditWhere(search string) (string, []any) {
	if search == "" {
		return "", nil
	}
	return " WHERE (u.email ILIKE $1 OR l.action ILIKE $1 OR l.target_type ILIKE $1)", []any{"%" + search + "%"}
}

func joinConditions(conditions []string) string {
	out := ""
	for i, condition := range conditions {
		if i > 0 {
			out += " AND "
		}
		out += condition
	}
	return out
}

type rowScanner interface {
	Scan(dest ...any) error
}

func queryOne(ctx context.Context, client agentQueryExecer, query string, args []any, dest ...any) error {
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return sql.ErrNoRows
	}
	if err := rows.Scan(dest...); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return rows.Err()
}

type agentQueryExecer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func scanAgentProfiles(rows *sql.Rows) ([]service.AgentProfile, error) {
	items := []service.AgentProfile{}
	for rows.Next() {
		item, err := scanAgentProfile(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func scanAgentProfile(row rowScanner) (*service.AgentProfile, error) {
	var p service.AgentProfile
	var parentID sql.NullInt64
	var parentUsername, parentEmail sql.NullString
	if err := row.Scan(
		&p.ID,
		&p.UserID,
		&p.Username,
		&p.Email,
		&p.Level,
		&parentID,
		&parentUsername,
		&parentEmail,
		&p.Status,
		&p.RateBPS,
		&p.ChildrenCount,
		&p.CustomersCount,
		&p.PayableAmount,
		&p.FrozenAmount,
		&p.CreatedAt,
		&p.DisabledAt,
	); err != nil {
		return nil, fmt.Errorf("scan agent profile: %w", err)
	}
	if parentID.Valid {
		p.ParentAgentID = &parentID.Int64
	}
	if parentUsername.Valid && parentUsername.String != "" {
		p.ParentUsername = &parentUsername.String
	}
	if parentEmail.Valid && parentEmail.String != "" {
		p.ParentEmail = &parentEmail.String
	}
	return &p, nil
}

func scanAgentCustomers(rows *sql.Rows) ([]service.AgentCustomer, error) {
	items := []service.AgentCustomer{}
	for rows.Next() {
		item, err := scanAgentCustomer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func scanAgentCustomer(row rowScanner) (*service.AgentCustomer, error) {
	var c service.AgentCustomer
	var sourceCode, subName sql.NullString
	var periodEnd sql.NullTime
	if err := row.Scan(&c.ID, &c.UserID, &c.Email, &c.Username, &c.Source, &sourceCode, &c.AgentID, &c.AgentName, &subName, &periodEnd, &c.ConfirmedRevenue, &c.Status); err != nil {
		return nil, fmt.Errorf("scan agent customer: %w", err)
	}
	if sourceCode.Valid {
		c.SourceReferralCode = &sourceCode.String
	}
	if subName.Valid {
		c.SubscriptionName = &subName.String
	}
	if periodEnd.Valid {
		c.PeriodEndAt = &periodEnd.Time
	}
	return &c, nil
}

func scanAgentCommissions(rows *sql.Rows) ([]service.AgentCommission, error) {
	items := []service.AgentCommission{}
	for rows.Next() {
		var c service.AgentCommission
		var orderID sql.NullInt64
		var reason sql.NullString
		var frozenUntil sql.NullTime
		if err := rows.Scan(&c.ID, &c.CustomerEmail, &orderID, &c.PeriodStartAt, &c.PeriodEndAt, &c.OrderPaidAmount, &c.ConfirmedRevenue, &c.RateBPS, &c.CommissionAmount, &c.ReverseAmount, &reason, &c.Status, &frozenUntil); err != nil {
			return nil, fmt.Errorf("scan agent commission: %w", err)
		}
		if orderID.Valid {
			c.OrderID = &orderID.Int64
		}
		if reason.Valid {
			c.ReverseReasonType = &reason.String
		}
		if frozenUntil.Valid {
			c.FrozenUntil = &frozenUntil.Time
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func scanAgentSettlements(rows *sql.Rows) ([]service.AgentSettlement, error) {
	items := []service.AgentSettlement{}
	for rows.Next() {
		item, err := scanAgentSettlement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func scanAgentSettlement(row rowScanner) (*service.AgentSettlement, error) {
	var s service.AgentSettlement
	var frozenUntil, paidAt sql.NullTime
	var paymentRef sql.NullString
	if err := row.Scan(&s.ID, &s.AgentID, &s.AgentEmail, &s.PeriodMonth, &s.Amount, &s.ReverseAmount, &s.NetAmount, &s.Status, &frozenUntil, &paidAt, &paymentRef); err != nil {
		return nil, fmt.Errorf("scan agent settlement: %w", err)
	}
	if frozenUntil.Valid {
		s.FrozenUntil = &frozenUntil.Time
	}
	if paidAt.Valid {
		s.PaidAt = &paidAt.Time
	}
	if paymentRef.Valid {
		s.PaymentReference = &paymentRef.String
	}
	return &s, nil
}

func scanAgentAuditLogs(rows *sql.Rows) ([]service.AgentAuditLog, error) {
	items := []service.AgentAuditLog{}
	for rows.Next() {
		var l service.AgentAuditLog
		var operatorID sql.NullInt64
		var reason sql.NullString
		if err := rows.Scan(&l.ID, &operatorID, &l.OperatorEmail, &l.OperatorRole, &l.Action, &l.TargetType, &l.TargetID, &reason, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan agent audit log: %w", err)
		}
		if operatorID.Valid {
			l.OperatorID = &operatorID.Int64
		}
		if reason.Valid {
			l.Reason = &reason.String
		}
		items = append(items, l)
	}
	return items, rows.Err()
}

func nullableInt64Value(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullablePositiveInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func nullableStringValue(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableJSON(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal audit json: %w", err)
	}
	return string(data), nil
}
