package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
)

type Migration struct {
	Filename string
	SQL      string
}

var migrations = []Migration{
	{
		Filename: "001_agent_admin_schema.sql",
		SQL: `
CREATE TABLE IF NOT EXISTS agent_admin_schema_migrations (
    filename TEXT PRIMARY KEY,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agent_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    level SMALLINT NOT NULL,
    parent_agent_id BIGINT NULL REFERENCES agent_profiles(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    contact_info JSONB NOT NULL DEFAULT '{}',
    settlement_info JSONB NOT NULL DEFAULT '{}',
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    disabled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_agent_profiles_level CHECK (level IN (1, 2, 3)),
    CONSTRAINT chk_agent_profiles_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_agent_profiles_parent CHECK (
        (level = 1 AND parent_agent_id IS NULL)
        OR (level IN (2, 3) AND parent_agent_id IS NOT NULL)
    )
);
CREATE INDEX IF NOT EXISTS idx_agent_profiles_parent_agent_id ON agent_profiles(parent_agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_profiles_status ON agent_profiles(status);
CREATE INDEX IF NOT EXISTS idx_agent_profiles_level ON agent_profiles(level);

CREATE TABLE IF NOT EXISTS agent_commission_rates (
    id BIGSERIAL PRIMARY KEY,
    agent_id BIGINT NOT NULL REFERENCES agent_profiles(id) ON DELETE CASCADE,
    rate_bps INT NOT NULL,
    set_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    set_by_agent_id BIGINT NULL REFERENCES agent_profiles(id) ON DELETE SET NULL,
    effective_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_agent_commission_rates_bps CHECK (rate_bps >= 0 AND rate_bps <= 10000)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_commission_rates_active_uniq
    ON agent_commission_rates(agent_id) WHERE expired_at IS NULL;
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
    ON agent_customer_relations(customer_user_id) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_agent_customer_relations_agent_id ON agent_customer_relations(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_customer_relations_customer_user_id ON agent_customer_relations(customer_user_id);

CREATE TABLE IF NOT EXISTS agent_customer_relation_changes (
    id BIGSERIAL PRIMARY KEY,
    customer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_agent_id BIGINT NULL REFERENCES agent_profiles(id) ON DELETE SET NULL,
    to_agent_id BIGINT NULL REFERENCES agent_profiles(id) ON DELETE SET NULL,
    reason TEXT NOT NULL,
    operator_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    effective_at TIMESTAMPTZ NOT NULL,
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
    rate_bps INT NOT NULL,
    commission_amount BIGINT NOT NULL DEFAULT 0,
    reverse_amount BIGINT NOT NULL DEFAULT 0,
    reverse_reason_type VARCHAR(64) NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'frozen',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    frozen_until TIMESTAMPTZ NULL,
    settlement_id BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_settlements_agent_month_uniq
    ON agent_settlements(agent_id, period_month);
CREATE INDEX IF NOT EXISTS idx_agent_settlements_status ON agent_settlements(status);

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
CREATE INDEX IF NOT EXISTS idx_agent_audit_logs_created_at ON agent_audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_audit_logs_target ON agent_audit_logs(target_type, target_id);
`,
	},
	{
		Filename: "002_agent_commission_period_unique_indexes.sql",
		SQL: `
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_commission_periods_order_agent_uniq
    ON agent_commission_periods(order_id, agent_id) WHERE order_id IS NOT NULL;
`,
	},
	{
		Filename: "003_agent_customer_relation_scheduled_status.sql",
		SQL: `
ALTER TABLE agent_customer_relations
    DROP CONSTRAINT IF EXISTS chk_agent_customer_relations_status;

ALTER TABLE agent_customer_relations
    ADD CONSTRAINT chk_agent_customer_relations_status
    CHECK (status IN ('active', 'scheduled', 'expired'));

CREATE INDEX IF NOT EXISTS idx_agent_customer_relations_scheduled_effective_at
    ON agent_customer_relations(effective_at)
    WHERE status = 'scheduled';
`,
	},
	{
		Filename: "004_agent_admin_settings.sql",
		SQL: `
CREATE TABLE IF NOT EXISTS agent_admin_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO agent_admin_settings (key, value)
VALUES
    ('turnstile_enabled', 'false'),
    ('turnstile_site_key', '')
ON CONFLICT (key) DO NOTHING;
`,
	},
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Filename < migrations[j].Filename
	})

	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS agent_admin_schema_migrations (
    filename TEXT PRIMARY KEY,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}

	for _, migration := range migrations {
		checksum := checksumSQL(migration.SQL)
		var existing string
		err := db.QueryRowContext(ctx, `SELECT checksum FROM agent_admin_schema_migrations WHERE filename = $1`, migration.Filename).Scan(&existing)
		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migration %s checksum mismatch", migration.Filename)
			}
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("query migration %s: %w", migration.Filename, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", migration.Filename, err)
		}
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", migration.Filename, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO agent_admin_schema_migrations (filename, checksum) VALUES ($1, $2)`, migration.Filename, checksum); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", migration.Filename, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", migration.Filename, err)
		}
	}

	return nil
}

func checksumSQL(sqlText string) string {
	sum := sha256.Sum256([]byte(sqlText))
	return hex.EncodeToString(sum[:])
}
