-- TTDC public alarm rule tables.
-- Run this script against the default configuration database, and create the
-- binding table in every business tenant database. It is safe to re-run.

CREATE TABLE IF NOT EXISTS ttdc_alarm_rule (
    id BIGSERIAL PRIMARY KEY,
    template_group_id BIGINT NOT NULL,
    template_id BIGINT NOT NULL,
    template_point_id BIGINT NOT NULL,
    base_data_id VARCHAR(48) NOT NULL,
    rule_name VARCHAR(128) NOT NULL,
    rule_code VARCHAR(128) NOT NULL,
    alarm_type VARCHAR(8) NOT NULL,
    alarm_stage INTEGER NOT NULL DEFAULT 1,
    alarm_channel VARCHAR(64),
    alarm_users VARCHAR(512),
    trigger_model INTEGER NOT NULL DEFAULT 0,
    formula VARCHAR(256) NOT NULL DEFAULT 'x',
    cycle BIGINT NOT NULL DEFAULT 0,
    sustain_time BIGINT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    version INTEGER NOT NULL DEFAULT 1,
    sort_no INTEGER NOT NULL DEFAULT 0,
    description VARCHAR(512),
    created_by INTEGER NOT NULL DEFAULT 0,
    updated_by INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    CONSTRAINT fk_ttdc_alarm_rule_group FOREIGN KEY (template_group_id) REFERENCES ttdc_template_group(id) ON DELETE RESTRICT,
    CONSTRAINT fk_ttdc_alarm_rule_template FOREIGN KEY (template_id) REFERENCES ttdc_template(id) ON DELETE RESTRICT,
    CONSTRAINT fk_ttdc_alarm_rule_point FOREIGN KEY (template_point_id) REFERENCES ttdc_template_point(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS ttdc_alarm_rule_logic (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT NOT NULL,
    logic VARCHAR(8) NOT NULL,
    trigger_value DOUBLE PRECISION NOT NULL DEFAULT 0,
    match_value VARCHAR(256),
    sort_no INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT fk_ttdc_alarm_rule_logic_rule FOREIGN KEY (rule_id) REFERENCES ttdc_alarm_rule(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ttdc_alarm_trigger_binding (
    id BIGSERIAL PRIMARY KEY,
    alarm_rule_id BIGINT NOT NULL,
    template_group_id BIGINT NOT NULL,
    general_trigger_id BIGINT NOT NULL,
    rule_version INTEGER NOT NULL DEFAULT 1,
    mark_hash VARCHAR(64),
    sync_status VARCHAR(16) NOT NULL DEFAULT 'ready',
    sync_error VARCHAR(512),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    CONSTRAINT uk_ttdc_alarm_trigger_binding UNIQUE (alarm_rule_id, template_group_id)
);

CREATE INDEX IF NOT EXISTS idx_ttdc_alarm_rule_group ON ttdc_alarm_rule(template_group_id, enabled, sort_no);
CREATE INDEX IF NOT EXISTS idx_ttdc_alarm_rule_point ON ttdc_alarm_rule(template_group_id, template_id, template_point_id);
CREATE INDEX IF NOT EXISTS idx_ttdc_alarm_rule_logic_rule ON ttdc_alarm_rule_logic(rule_id, sort_no);
CREATE INDEX IF NOT EXISTS idx_ttdc_alarm_trigger_binding_trigger ON ttdc_alarm_trigger_binding(general_trigger_id);

-- OpenGauss does not support PostgreSQL's ADD COLUMN IF NOT EXISTS syntax.
-- Keep this idempotent for databases that already have the binding table.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = CURRENT_SCHEMA()
                 AND table_name = 'ttdc_alarm_trigger_binding')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                       WHERE table_schema = CURRENT_SCHEMA()
                         AND table_name = 'ttdc_alarm_trigger_binding'
                         AND column_name = 'mark_hash') THEN
        ALTER TABLE ttdc_alarm_trigger_binding ADD COLUMN mark_hash VARCHAR(64);
    END IF;
END $$;
