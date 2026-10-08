package migration

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// EnsureBusinessRecordTimeColumns 在 AutoMigrate 之前兼容旧模型误建的 JSON 时间列。
// 已有 bigint 不做修改；遇到非整数值立即回滚，不删除记录、不将异常时间悄悄置零。
func EnsureBusinessRecordTimeColumns(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("业务记录迁移数据库连接为空")
	}
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	// 表名只取代码内固定白名单，不使用外部输入构造数据库标识符。
	for _, table := range []string{"equipment_maintain_record", "equipment_circulation_log", "equipment_geography_position"} {
		if err := migrateBusinessRecordTimeColumn(db, table); err != nil {
			return fmt.Errorf("迁移 %s 的创建时间失败: %w", table, err)
		}
	}
	return nil
}

func migrateBusinessRecordTimeColumn(db *gorm.DB, table string) error {
	columnType, err := maintainRecordTimeColumnType(db, table)
	if err != nil || (columnType != "json" && columnType != "jsonb") {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// 多服务可能同时启动，先锁定表再检查类型，避免重复转换。
		if err := tx.Exec("LOCK TABLE public." + table + " IN ACCESS EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		columnType, err := maintainRecordTimeColumnType(tx, table)
		if err != nil || (columnType != "json" && columnType != "jsonb") {
			return err
		}
		rows, err := tx.Raw("SELECT id, create_time::text FROM public." + table + " WHERE create_time IS NOT NULL").Rows()
		if err != nil {
			return err
		}
		for rows.Next() {
			var id uint64
			var raw string
			if err := rows.Scan(&id, &raw); err != nil {
				rows.Close()
				return err
			}
			if err := validateMaintainRecordTimeJSON(raw); err != nil {
				rows.Close()
				return fmt.Errorf("记录 %d 的创建时间不是有效整数，已保留原数据并停止迁移: %w", id, err)
			}
		}
		iterationErr := rows.Err()
		rows.Close()
		if iterationErr != nil {
			return iterationErr
		}
		// JSON 数字及旧版数字字符串均无损转换；SQL NULL 和 JSON null 都保留空值。
		return tx.Exec(`ALTER TABLE public.` + table + ` ALTER COLUMN create_time TYPE bigint
			USING CASE WHEN create_time IS NULL OR create_time::text = 'null' THEN NULL
			ELSE trim(both '"' from create_time::text)::bigint END`).Error
	})
}

func maintainRecordTimeColumnType(db *gorm.DB, table string) (string, error) {
	var columnType string
	err := db.Raw(`SELECT data_type FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ? AND column_name = 'create_time'`, table).Scan(&columnType).Error
	return columnType, err
}

func validateMaintainRecordTimeJSON(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "null" {
		return nil
	}
	if !json.Valid([]byte(raw)) {
		return fmt.Errorf("JSON 内容无效")
	}
	value := raw
	if strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`) {
		value = raw[1 : len(raw)-1]
	}
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return fmt.Errorf("需要 64 位整数秒级时间戳")
	}
	return nil
}
