package app

import (
	"context"
	"log"
	"time"
)

type Scheduler struct {
	cfg      Config
	repo     *Repository
	interval time.Duration
	stop     chan struct{}
}

func NewScheduler(cfg Config, repo *Repository) *Scheduler {
	return &Scheduler{
		cfg:      cfg,
		repo:     repo,
		interval: 24 * time.Hour,
		stop:     make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	if s == nil || s.repo == nil {
		return
	}
	go func() {
		s.runOnce()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stop:
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	if s == nil {
		return
	}
	close(s.stop)
}

func (s *Scheduler) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := s.repo.SyncReferralCustomerRelations(ctx); err != nil {
		log.Printf("sync referral customer relations failed: %v", err)
	}
	if err := s.repo.ActivateScheduledCustomerRelations(ctx); err != nil {
		log.Printf("activate scheduled customer relations failed: %v", err)
	}
	if err := s.repo.GenerateCommissionPeriods(ctx, s.cfg); err != nil {
		log.Printf("generate commission periods failed: %v", err)
	}
	if err := s.repo.GenerateSettlements(ctx, s.cfg); err != nil {
		log.Printf("generate settlements failed: %v", err)
	}
}

func (r *Repository) SyncReferralCustomerRelations(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO agent_customer_relations (
    agent_id, customer_user_id, source, source_referral_code, effective_at, status, created_by
)
SELECT
    ap.id,
    ua.user_id,
    'referral',
    inviter_aff.aff_code,
    ua.created_at,
    'active',
    ap.user_id
FROM user_affiliates ua
JOIN agent_profiles ap ON ap.user_id = ua.inviter_id
LEFT JOIN user_affiliates inviter_aff ON inviter_aff.user_id = ua.inviter_id
WHERE ua.inviter_id IS NOT NULL
  AND ap.status = 'active'
  AND ua.user_id <> ap.user_id
  AND NOT EXISTS (
      SELECT 1
      FROM agent_customer_relations existing
      WHERE existing.customer_user_id = ua.user_id
        AND existing.status = 'active'
  )
ON CONFLICT DO NOTHING`)
	return err
}

func (r *Repository) GenerateCommissionPeriods(ctx context.Context, cfg Config) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
WITH RECURSIVE eligible_orders AS (
    SELECT
        po.id AS order_id,
        po.user_id AS customer_user_id,
        us.id AS subscription_id,
        us.starts_at AS period_start_at,
        us.expires_at AS period_end_at,
        ROUND(GREATEST(COALESCE(po.pay_amount, po.amount, 0) - COALESCE(po.refund_amount, 0), 0) * 100)::bigint AS confirmed_revenue
    FROM payment_orders po
    JOIN user_subscriptions us
      ON us.user_id = po.user_id
     AND us.group_id = po.subscription_group_id
     AND us.deleted_at IS NULL
    WHERE po.order_type = 'subscription'
      AND po.status IN ('PAID', 'COMPLETED')
      AND po.subscription_group_id IS NOT NULL
      AND po.paid_at IS NOT NULL
      AND us.expires_at <= NOW()
      AND COALESCE(po.completed_at, po.paid_at) <= us.expires_at
      AND po.paid_at >= us.starts_at
      AND po.paid_at <= us.expires_at
),
base_relations AS (
    SELECT DISTINCT ON (eo.order_id)
        eo.*,
        acr.agent_id AS direct_agent_id
    FROM eligible_orders eo
    JOIN agent_customer_relations acr
      ON acr.customer_user_id = eo.customer_user_id
     AND acr.status = 'active'
     AND acr.effective_at <= eo.period_start_at
    WHERE NOT EXISTS (
        SELECT 1
        FROM agent_commission_periods existing
        WHERE existing.order_id = eo.order_id
    )
    ORDER BY eo.order_id, acr.effective_at DESC, acr.id DESC
),
chain AS (
    SELECT
        br.*,
        ap.id AS agent_id,
        ap.parent_agent_id,
        ap.level
    FROM base_relations br
    JOIN agent_profiles ap ON ap.id = br.direct_agent_id
    WHERE ap.status = 'active'

    UNION ALL

    SELECT
        c.order_id,
        c.customer_user_id,
        c.subscription_id,
        c.period_start_at,
        c.period_end_at,
        c.confirmed_revenue,
        c.direct_agent_id,
        parent.id,
        parent.parent_agent_id,
        parent.level
    FROM chain c
    JOIN agent_profiles parent ON parent.id = c.parent_agent_id
    WHERE parent.status = 'active'
),
rated AS (
    SELECT
        c.*,
        COALESCE(own_rate.rate_bps, 0) AS own_rate_bps,
        COALESCE(child_rate.rate_bps, 0) AS child_rate_bps
    FROM chain c
    LEFT JOIN LATERAL (
        SELECT rate_bps
        FROM agent_commission_rates r
        WHERE r.agent_id = c.agent_id
          AND r.effective_at <= c.period_start_at
          AND (r.expired_at IS NULL OR r.expired_at > c.period_start_at)
        ORDER BY r.effective_at DESC, r.id DESC
        LIMIT 1
    ) own_rate ON true
    LEFT JOIN LATERAL (
        SELECT rate_bps
        FROM agent_commission_rates r
        WHERE r.agent_id = (
            SELECT child.id
            FROM chain child
            WHERE child.order_id = c.order_id
              AND child.parent_agent_id = c.agent_id
            LIMIT 1
        )
          AND r.effective_at <= c.period_start_at
          AND (r.expired_at IS NULL OR r.expired_at > c.period_start_at)
        ORDER BY r.effective_at DESC, r.id DESC
        LIMIT 1
    ) child_rate ON true
),
amounts AS (
    SELECT
        order_id,
        customer_user_id,
        agent_id,
        subscription_id,
        period_start_at,
        period_end_at,
        confirmed_revenue,
        own_rate_bps,
        GREATEST(own_rate_bps - child_rate_bps, 0) AS effective_rate_bps,
        ROUND(confirmed_revenue * GREATEST(own_rate_bps - child_rate_bps, 0)::numeric / 10000)::bigint AS commission_amount
    FROM rated
)
INSERT INTO agent_commission_periods (
    customer_user_id, agent_id, order_id, subscription_id, period_start_at, period_end_at,
    order_paid_amount, confirmed_revenue, rate_bps, commission_amount, status, frozen_until
)
SELECT
    customer_user_id,
    agent_id,
    order_id,
    subscription_id,
    period_start_at,
    period_end_at,
    confirmed_revenue,
    confirmed_revenue,
    effective_rate_bps,
    commission_amount,
    'frozen',
    period_end_at + ($1::int * INTERVAL '1 day')
FROM amounts
WHERE confirmed_revenue > 0
  AND commission_amount > 0
ON CONFLICT DO NOTHING`, cfg.FreezeDays); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE agent_commission_periods
SET status = 'payable', updated_at = NOW()
WHERE status = 'frozen'
  AND frozen_until IS NOT NULL
  AND frozen_until <= NOW()`); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GenerateSettlements(ctx context.Context, cfg Config) error {
	loc := cfg.Location()
	now := time.Now().In(loc)
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
WITH monthly AS (
    SELECT
        agent_id,
        date_trunc('month', period_end_at AT TIME ZONE $1)::date AS period_month,
        COALESCE(SUM(commission_amount), 0)::bigint AS amount,
        COALESCE(SUM(reverse_amount), 0)::bigint AS reverse_amount,
        MAX(frozen_until) AS frozen_until
    FROM agent_commission_periods
    WHERE status IN ('frozen', 'payable')
      AND date_trunc('month', period_end_at AT TIME ZONE $1)::date < $2::date
    GROUP BY agent_id, date_trunc('month', period_end_at AT TIME ZONE $1)::date
)
INSERT INTO agent_settlements (
    agent_id, period_month, amount, reverse_amount, net_amount, status, min_amount_met, frozen_until
)
SELECT
    agent_id,
    period_month,
    amount,
    reverse_amount,
    amount - reverse_amount,
    CASE
      WHEN amount - reverse_amount >= $3::bigint AND COALESCE(frozen_until, NOW()) <= NOW() THEN 'payable'
      ELSE 'pending'
    END,
    amount - reverse_amount >= $3::bigint,
    frozen_until
FROM monthly
ON CONFLICT (agent_id, period_month) DO UPDATE
SET amount = EXCLUDED.amount,
    reverse_amount = EXCLUDED.reverse_amount,
    net_amount = EXCLUDED.net_amount,
    status = CASE
      WHEN agent_settlements.status = 'paid' THEN agent_settlements.status
      ELSE EXCLUDED.status
    END,
    min_amount_met = EXCLUDED.min_amount_met,
    frozen_until = EXCLUDED.frozen_until,
    updated_at = NOW()`, cfg.SettlementTimezone, currentMonth, cfg.MinSettlementAmount); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE agent_commission_periods cp
SET settlement_id = s.id,
    updated_at = NOW()
FROM agent_settlements s
WHERE cp.agent_id = s.agent_id
  AND date_trunc('month', cp.period_end_at AT TIME ZONE $1)::date = s.period_month
  AND cp.settlement_id IS NULL
  AND cp.status IN ('frozen', 'payable')`, cfg.SettlementTimezone); err != nil {
		return err
	}

	return tx.Commit()
}
