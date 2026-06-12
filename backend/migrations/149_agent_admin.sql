CREATE TABLE IF NOT EXISTS agent_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    level SMALLINT NOT NULL,
    parent_agent_id BIGINT NULL REFERENCES agent_profiles(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    contact_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    settlement_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    disabled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_agent_profiles_level CHECK (level IN (1, 2, 3)),
    CONSTRAINT chk_agent_profiles_status CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_parent_agent_id ON agent_profiles(parent_agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_profiles_status ON agent_profiles(status);
CREATE INDEX IF NOT EXISTS idx_agent_profiles_level ON agent_profiles(level);

CREATE TABLE IF NOT EXISTS agent_commission_rates (
    id BIGSERIAL PRIMARY KEY,
    agent_id BIGINT NOT NULL REFERENCES agent_profiles(id) ON DELETE CASCADE,
    rate_bps INTEGER NOT NULL,
    set_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    set_by_agent_id BIGINT NULL REFERENCES agent_profiles(id) ON DELETE SET NULL,
    effective_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_agent_commission_rates_bps CHECK (rate_bps >= 0 AND rate_bps <= 10000)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_commission_rates_active_uniq
ON agent_commission_rates(agent_id)
WHERE expired_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_commission_rates_agent_time
ON agent_commission_rates(agent_id, effective_at DESC);

CREATE TABLE IF NOT EXISTS agent_customer_relations (
    id BIGSERIAL PRIMARY KEY,
    agent_id BIGINT NOT NULL REFERENCES agent_profiles(id) ON DELETE CASCADE,
    customer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source VARCHAR(20) NOT NULL,
    source_referral_code VARCHAR(32) NULL,
    effective_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at TIMESTAMPTZ NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_agent_customer_relations_source CHECK (source IN ('referral', 'manual')),
    CONSTRAINT chk_agent_customer_relations_status CHECK (status IN ('active', 'expired'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_customer_relations_active_customer_uniq
ON agent_customer_relations(customer_user_id)
WHERE status = 'active' AND expired_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_customer_relations_agent_id ON agent_customer_relations(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_customer_relations_customer_user_id ON agent_customer_relations(customer_user_id);

CREATE TABLE IF NOT EXISTS agent_customer_relation_changes (
    id BIGSERIAL PRIMARY KEY,
    customer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_agent_id BIGINT NULL REFERENCES agent_profiles(id) ON DELETE SET NULL,
    to_agent_id BIGINT NULL REFERENCES agent_profiles(id) ON DELETE SET NULL,
    reason TEXT NOT NULL,
    operator_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    effective_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_customer_relation_changes_customer_user_id
ON agent_customer_relation_changes(customer_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_commission_periods (
    id BIGSERIAL PRIMARY KEY,
    customer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id BIGINT NOT NULL REFERENCES agent_profiles(id) ON DELETE CASCADE,
    order_id BIGINT NULL,
    subscription_id BIGINT NULL,
    period_start_at TIMESTAMPTZ NOT NULL,
    period_end_at TIMESTAMPTZ NOT NULL,
    order_paid_amount BIGINT NOT NULL DEFAULT 0,
    confirmed_revenue BIGINT NOT NULL DEFAULT 0,
    rate_bps INTEGER NOT NULL DEFAULT 0,
    commission_amount BIGINT NOT NULL DEFAULT 0,
    reverse_amount BIGINT NOT NULL DEFAULT 0,
    reverse_reason_type VARCHAR(64) NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    frozen_until TIMESTAMPTZ NULL,
    settlement_id BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_agent_commission_periods_status CHECK (status IN ('pending', 'frozen', 'payable', 'paid', 'reversed')),
    CONSTRAINT chk_agent_commission_periods_rate CHECK (rate_bps >= 0 AND rate_bps <= 10000)
);

CREATE INDEX IF NOT EXISTS idx_agent_commission_periods_agent_id ON agent_commission_periods(agent_id, period_end_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_commission_periods_customer_user_id ON agent_commission_periods(customer_user_id);
CREATE INDEX IF NOT EXISTS idx_agent_commission_periods_status ON agent_commission_periods(status);

CREATE TABLE IF NOT EXISTS agent_settlements (
    id BIGSERIAL PRIMARY KEY,
    agent_id BIGINT NOT NULL REFERENCES agent_profiles(id) ON DELETE CASCADE,
    period_month DATE NOT NULL,
    amount BIGINT NOT NULL DEFAULT 0,
    reverse_amount BIGINT NOT NULL DEFAULT 0,
    net_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    min_amount_met BOOLEAN NOT NULL DEFAULT FALSE,
    frozen_until TIMESTAMPTZ NULL,
    paid_at TIMESTAMPTZ NULL,
    paid_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    payment_reference VARCHAR(128) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_agent_settlements_status CHECK (status IN ('pending', 'frozen', 'payable', 'paid', 'reversed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_settlements_agent_month_uniq
ON agent_settlements(agent_id, period_month);

CREATE INDEX IF NOT EXISTS idx_agent_settlements_status ON agent_settlements(status);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_agent_commission_periods_settlement'
          AND conrelid = 'agent_commission_periods'::regclass
    ) THEN
        ALTER TABLE agent_commission_periods
        ADD CONSTRAINT fk_agent_commission_periods_settlement
        FOREIGN KEY (settlement_id) REFERENCES agent_settlements(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS agent_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    operator_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    operator_role VARCHAR(20) NOT NULL,
    action VARCHAR(64) NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    target_id BIGINT NOT NULL,
    before_data JSONB NULL,
    after_data JSONB NULL,
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_audit_logs_target ON agent_audit_logs(target_type, target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_audit_logs_operator_user_id ON agent_audit_logs(operator_user_id, created_at DESC);

COMMENT ON TABLE agent_profiles IS '代理商档案，与 users 共用账号体系';
COMMENT ON TABLE agent_commission_rates IS '代理商当前及历史分成比例，单位 bps';
COMMENT ON TABLE agent_customer_relations IS '客户归属代理关系，一个客户同一时间只能归属一个代理';
COMMENT ON TABLE agent_commission_periods IS '按套餐周期确认的代理分成记录，金额单位为分';
COMMENT ON TABLE agent_settlements IS '代理月度结算记录，金额单位为分';
COMMENT ON TABLE agent_audit_logs IS 'agent-admin 操作审计日志';
