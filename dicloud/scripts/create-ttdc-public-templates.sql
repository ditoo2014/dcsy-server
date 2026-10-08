-- TTDC public-template and automatic-channel schema verification helpers.
-- The authoritative staged migration is cmd/ttdcmigrate. Run that command
-- before enabling battery auto creation; this file is safe to re-run after
-- the migration has reached the ready state.

CREATE UNIQUE INDEX IF NOT EXISTS uk_ttdc_template_group_code
ON ttdc_template_group(group_code);

CREATE UNIQUE INDEX IF NOT EXISTS uk_ttdc_template_group_runtime_topic
ON ttdc_template_group(topic_segment)
WHERE deleted_at IS NULL AND runtime_match_enabled IS TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS uk_ttdc_template_code
ON ttdc_template(template_code);

CREATE UNIQUE INDEX IF NOT EXISTS uk_ttdc_template_group_role
ON ttdc_template(group_id, cid2_type)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_ttdc_template_protocol_code
ON ttdc_template(protocol_code);

CREATE UNIQUE INDEX IF NOT EXISTS uk_ttdc_template_point_group_base_id
ON ttdc_template_point(group_id, base_data_id);

CREATE UNIQUE INDEX IF NOT EXISTS uk_ttdc_template_point_sort_no
ON ttdc_template_point(template_id, sort_no)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_gateway_ttdc_address_group
ON gateway_ttdc_address_group_binding(gateway_id, module_adr);

CREATE UNIQUE INDEX IF NOT EXISTS uk_gateway_ttdc_channel_role
ON gateway_ttdc_template_binding(gateway_id, module_adr, protocol_code);

CREATE UNIQUE INDEX IF NOT EXISTS uk_gateway_ttdc_binding_module
ON gateway_ttdc_template_binding(module_id);

CREATE UNIQUE INDEX IF NOT EXISTS uk_gateway_data_gateway_data_id
ON gateway_data(gateway_id, data_id);

CREATE UNIQUE INDEX IF NOT EXISTS uk_gateway_module_ttdc_data_module_data_id
ON gateway_module_ttdc_data(module_id, data_id);

CREATE UNIQUE INDEX IF NOT EXISTS uk_gateway_module_ttdc_data_algorithm_key
ON gateway_module_ttdc_data(module_id, algorithm_key)
WHERE algorithm_key IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_gateway_ttdc_address_group_template') THEN
        ALTER TABLE gateway_ttdc_address_group_binding
            ADD CONSTRAINT fk_gateway_ttdc_address_group_template
            FOREIGN KEY (template_group_id) REFERENCES ttdc_template_group(id)
            ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_gateway_ttdc_template_binding_group') THEN
        ALTER TABLE gateway_ttdc_template_binding
            ADD CONSTRAINT fk_gateway_ttdc_template_binding_group
            FOREIGN KEY (template_group_id) REFERENCES ttdc_template_group(id)
            ON DELETE RESTRICT;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_ttdc_config_revision_cursor
ON ttdc_config_revision(id, created_at);
