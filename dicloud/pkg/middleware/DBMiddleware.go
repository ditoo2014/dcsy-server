package middleware

import (
	"dicloud/util/opengauss"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DBMiddleware struct {
	db *gorm.DB
}

func NewDBMiddleware() *DBMiddleware {
	return &DBMiddleware{
		db: opengauss.GetGormDB(&gin.Context{}), // 使用 GORM 的 DB 实例
	}
}

// ExecuteQuery 执行查询语句
func (mw *DBMiddleware) ExecuteQuery(model interface{}, query string, args ...interface{}) error {
	result := mw.db.Raw(query, args...).Scan(model) // 使用 GORM 的 Raw 方法
	if result.Error != nil {
		return mw.logError("Failed to execute query", result.Error)
	}
	return nil
}

// ExecuteExec 执行修改语句（插入、更新、删除）
func (mw *DBMiddleware) ExecuteExec(query string, args ...interface{}) (int64, error) {
	result := mw.db.Exec(query, args...) // 使用 GORM 的 Exec 方法
	if result.Error != nil {
		return 0, mw.logError("Failed to execute exec", result.Error)
	}
	return result.RowsAffected, nil // 返回影响的行数
}

// BeginTransaction 开启事务
func (mw *DBMiddleware) BeginTransaction() (*gorm.DB, error) {
	tx := mw.db.Begin() // 使用 GORM 的事务支持
	if tx.Error != nil {
		return nil, mw.logError("Failed to begin transaction", tx.Error)
	}
	return tx, nil
}

// logError 记录错误信息
func (mw *DBMiddleware) logError(message string, err error) error {
	log.Printf("%s: %v", message, err)
	return err
}
