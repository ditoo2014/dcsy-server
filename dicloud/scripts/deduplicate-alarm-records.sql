-- Run once for the default database and for every tenant database.
-- Stop the alarm consumer before execution.

START TRANSACTION;

LOCK TABLE alarm_data_record IN SHARE ROW EXCLUSIVE MODE;

CREATE TABLE IF NOT EXISTS alarm_data_record_dedup_backup AS
SELECT alarm_data_record.*, CURRENT_TIMESTAMP AS deduplicated_at
FROM alarm_data_record
WHERE 1 = 0;

WITH duplicate_ids AS (
	SELECT id
	FROM (
		SELECT
			id,
			ROW_NUMBER() OVER (
				PARTITION BY equipment_id, alarm_code, begin_time, alarm_type, alarm_stage
				ORDER BY id ASC
			) AS row_number
		FROM alarm_data_record
	) ranked
	WHERE row_number > 1
)
INSERT INTO alarm_data_record_dedup_backup
SELECT alarm_data_record.*, CURRENT_TIMESTAMP
FROM alarm_data_record
JOIN duplicate_ids ON duplicate_ids.id = alarm_data_record.id
WHERE NOT EXISTS (
	SELECT 1
	FROM alarm_data_record_dedup_backup backup
	WHERE backup.id = alarm_data_record.id
);

WITH duplicate_ids AS (
	SELECT id
	FROM (
		SELECT
			id,
			ROW_NUMBER() OVER (
				PARTITION BY equipment_id, alarm_code, begin_time, alarm_type, alarm_stage
				ORDER BY id ASC
			) AS row_number
		FROM alarm_data_record
	) ranked
	WHERE row_number > 1
)
DELETE FROM alarm_data_record
USING duplicate_ids
WHERE alarm_data_record.id = duplicate_ids.id;

UPDATE equipment
SET alarm_trigger_num = CAST((
	SELECT COUNT(*)
	FROM alarm_data_record
	WHERE alarm_data_record.equipment_id = equipment.id
		AND alarm_data_record.alarm_type = '0'
		AND alarm_data_record.alarm_status = 1
) AS INTEGER);

DROP INDEX IF EXISTS uni_alarm_record_lifecycle;
CREATE UNIQUE INDEX uni_alarm_record_lifecycle
ON alarm_data_record (equipment_id, alarm_code, begin_time, alarm_type, alarm_stage);

COMMIT;

SELECT COUNT(*) AS remaining_duplicate_groups
FROM (
	SELECT 1
	FROM alarm_data_record
	GROUP BY equipment_id, alarm_code, begin_time, alarm_type, alarm_stage
	HAVING COUNT(*) > 1
) duplicate_groups;
