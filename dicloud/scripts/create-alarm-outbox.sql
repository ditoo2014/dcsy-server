-- Run for the default database and every tenant database before starting alarm.

START TRANSACTION;

CREATE TABLE IF NOT EXISTS alarm_event_inbox (
	id bigserial PRIMARY KEY,
	event_id varchar(128) NOT NULL,
	lifecycle_key varchar(64) NOT NULL,
	payload_hash varchar(64) NOT NULL,
	payload text NOT NULL,
	status varchar(24) NOT NULL,
	alarm_record_id bigint NOT NULL DEFAULT 0,
	conflict_hash varchar(64) NOT NULL DEFAULT '-',
	processed_at bigint NOT NULL DEFAULT 0,
	create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS alarm_notification_delivery (
	id bigserial PRIMARY KEY,
	event_id varchar(128) NOT NULL,
	lifecycle_key varchar(64) NOT NULL,
	alarm_record_id bigint NOT NULL DEFAULT 0,
	channel varchar(32) NOT NULL,
	recipient_key varchar(80) NOT NULL,
	recipient_ids text NOT NULL DEFAULT '[]',
	alarm_payload text NOT NULL,
	status varchar(32) NOT NULL,
	attempt_count integer NOT NULL DEFAULT 0,
	max_attempts integer NOT NULL DEFAULT 5,
	next_retry_at bigint NOT NULL DEFAULT 0,
	lease_owner varchar(128) NOT NULL DEFAULT '-',
	lease_until bigint NOT NULL DEFAULT 0,
	provider_request_id varchar(128) NOT NULL DEFAULT '-',
	response_code varchar(128) NOT NULL DEFAULT '-',
	response text NOT NULL DEFAULT '-',
	last_error text NOT NULL DEFAULT '-',
	sent_at bigint NOT NULL DEFAULT 0,
	create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Oracle-compatible OpenGauss treats an empty string as NULL. Keep optional
-- audit columns non-empty so NOT NULL works in every compatibility mode.
ALTER TABLE alarm_event_inbox ALTER COLUMN conflict_hash SET DEFAULT '-';
UPDATE alarm_event_inbox SET conflict_hash = '-'
WHERE COALESCE(length(conflict_hash), 0) = 0;
ALTER TABLE alarm_event_inbox ALTER COLUMN conflict_hash SET NOT NULL;

ALTER TABLE alarm_notification_delivery ALTER COLUMN lease_owner SET DEFAULT '-';
ALTER TABLE alarm_notification_delivery ALTER COLUMN provider_request_id SET DEFAULT '-';
ALTER TABLE alarm_notification_delivery ALTER COLUMN response_code SET DEFAULT '-';
ALTER TABLE alarm_notification_delivery ALTER COLUMN response SET DEFAULT '-';
ALTER TABLE alarm_notification_delivery ALTER COLUMN last_error SET DEFAULT '-';

UPDATE alarm_notification_delivery SET lease_owner = '-'
WHERE COALESCE(length(lease_owner), 0) = 0;
UPDATE alarm_notification_delivery SET provider_request_id = '-'
WHERE COALESCE(length(provider_request_id), 0) = 0;
UPDATE alarm_notification_delivery SET response_code = '-'
WHERE COALESCE(length(response_code), 0) = 0;
UPDATE alarm_notification_delivery SET response = '-'
WHERE COALESCE(length(response), 0) = 0;
UPDATE alarm_notification_delivery SET last_error = '-'
WHERE COALESCE(length(last_error), 0) = 0;

ALTER TABLE alarm_notification_delivery ALTER COLUMN lease_owner SET NOT NULL;
ALTER TABLE alarm_notification_delivery ALTER COLUMN provider_request_id SET NOT NULL;
ALTER TABLE alarm_notification_delivery ALTER COLUMN response_code SET NOT NULL;
ALTER TABLE alarm_notification_delivery ALTER COLUMN response SET NOT NULL;
ALTER TABLE alarm_notification_delivery ALTER COLUMN last_error SET NOT NULL;

DROP INDEX IF EXISTS uni_alarm_event_inbox_event;
CREATE UNIQUE INDEX uni_alarm_event_inbox_event ON alarm_event_inbox (event_id);

DROP INDEX IF EXISTS uni_alarm_delivery_target;
CREATE UNIQUE INDEX uni_alarm_delivery_target
ON alarm_notification_delivery (event_id, channel, recipient_key);

DROP INDEX IF EXISTS idx_alarm_delivery_due;
CREATE INDEX idx_alarm_delivery_due
ON alarm_notification_delivery (status, next_retry_at, lease_until);

DROP INDEX IF EXISTS idx_alarm_delivery_lifecycle;
CREATE INDEX idx_alarm_delivery_lifecycle
ON alarm_notification_delivery (lifecycle_key, id, status);

COMMIT;
