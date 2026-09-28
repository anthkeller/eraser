package history

import (
	"fmt"
	"strings"
)

func (s *Store) migratePostgres() error {
	query := `
	CREATE TABLE IF NOT EXISTS removal_requests (
		id BIGSERIAL PRIMARY KEY,
		owner_id TEXT NOT NULL DEFAULT current_setting('app.owner_id'),
		broker_id TEXT NOT NULL,
		broker_name TEXT NOT NULL,
		email TEXT NOT NULL,
		template TEXT NOT NULL,
		status TEXT NOT NULL,
		message_id TEXT,
		error TEXT,
		sent_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		pipeline_status TEXT DEFAULT 'email_sent'
	);
	CREATE INDEX IF NOT EXISTS idx_rr_owner_broker ON removal_requests(owner_id, broker_id);
	CREATE INDEX IF NOT EXISTS idx_rr_owner_sent ON removal_requests(owner_id, sent_at);

	CREATE TABLE IF NOT EXISTS broker_responses (
		id BIGSERIAL PRIMARY KEY,
		owner_id TEXT NOT NULL DEFAULT current_setting('app.owner_id'),
		broker_id TEXT NOT NULL,
		broker_name TEXT NOT NULL,
		response_type TEXT NOT NULL,
		email_from TEXT,
		email_subject TEXT,
		email_body TEXT,
		form_url TEXT,
		confirm_url TEXT,
		confidence DOUBLE PRECISION,
		needs_review INTEGER DEFAULT 0,
		received_at TIMESTAMPTZ,
		processed_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_br_owner_broker ON broker_responses(owner_id, broker_id);

	CREATE TABLE IF NOT EXISTS pending_tasks (
		id BIGSERIAL PRIMARY KEY,
		owner_id TEXT NOT NULL DEFAULT current_setting('app.owner_id'),
		broker_id TEXT NOT NULL,
		broker_name TEXT NOT NULL,
		task_type TEXT NOT NULL,
		form_url TEXT,
		screenshot_path TEXT,
		browser_state TEXT,
		notes TEXT,
		status TEXT DEFAULT 'pending',
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		opened_at TIMESTAMPTZ,
		completed_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_pt_owner_status ON pending_tasks(owner_id, status);

	CREATE TABLE IF NOT EXISTS exposure_checks (
		id BIGSERIAL PRIMARY KEY,
		owner_id TEXT NOT NULL DEFAULT current_setting('app.owner_id'),
		broker_id TEXT NOT NULL,
		broker_name TEXT NOT NULL,
		status TEXT NOT NULL,
		record_url TEXT,
		evidence TEXT,
		confidence DOUBLE PRECISION DEFAULT 0,
		error TEXT,
		checked_at TIMESTAMPTZ NOT NULL,
		next_check_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_ec_owner_broker ON exposure_checks(owner_id, broker_id);
	CREATE INDEX IF NOT EXISTS idx_ec_owner_next ON exposure_checks(owner_id, next_check_at);

	CREATE TABLE IF NOT EXISTS broker_validations (
		id BIGSERIAL PRIMARY KEY,
		owner_id TEXT NOT NULL DEFAULT current_setting('app.owner_id'),
		broker_id TEXT NOT NULL,
		website_valid BOOLEAN DEFAULT FALSE,
		website_status_code INTEGER DEFAULT 0,
		optout_valid BOOLEAN DEFAULT FALSE,
		optout_status_code INTEGER DEFAULT 0,
		email_domain_valid BOOLEAN DEFAULT FALSE,
		error TEXT,
		checked_at TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_bv_owner_broker ON broker_validations(owner_id, broker_id);

	CREATE TABLE IF NOT EXISTS account_inventory (
		id BIGSERIAL PRIMARY KEY,
		owner_id TEXT NOT NULL DEFAULT current_setting('app.owner_id'),
		service TEXT NOT NULL,
		login_url TEXT NOT NULL,
		username TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		privacy_url TEXT,
		imported_at TIMESTAMPTZ NOT NULL,
		UNIQUE(owner_id, login_url, username)
	);
	CREATE INDEX IF NOT EXISTS idx_ai_owner_status ON account_inventory(owner_id, status);
	`
	if err := s.execPostgresStatements(query); err != nil {
		return fmt.Errorf("migrate PostgreSQL database: %w", err)
	}

	for _, table := range []string{"removal_requests", "broker_responses", "pending_tasks", "exposure_checks", "broker_validations", "account_inventory"} {
		policy := table + "_owner_isolation"
		statement := fmt.Sprintf(`
			ALTER TABLE %s ENABLE ROW LEVEL SECURITY;
			ALTER TABLE %s FORCE ROW LEVEL SECURITY;
			DROP POLICY IF EXISTS %s ON %s;
			CREATE POLICY %s ON %s
			USING (owner_id = current_setting('app.owner_id'))
			WITH CHECK (owner_id = current_setting('app.owner_id'));`, table, table, policy, table, policy, table)
		if err := s.execPostgresStatements(statement); err != nil {
			return fmt.Errorf("enable owner isolation on %s: %w", table, err)
		}
	}
	return nil
}

func (s *Store) execPostgresStatements(script string) error {
	for _, statement := range strings.Split(script, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}
