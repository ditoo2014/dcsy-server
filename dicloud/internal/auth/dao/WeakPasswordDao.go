package dao

import (
	"fmt"
	"strings"

	"dicloud/util/opengauss"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WeakPasswordDAO struct {
	db *gorm.DB
}

// NewWeakPasswordDAO always uses the auth service's default database. The
// dictionary is global and must not be selected from a renter database.
func NewWeakPasswordDAO() *WeakPasswordDAO {
	return &WeakPasswordDAO{db: opengauss.GetGormDB(&gin.Context{})}
}

func NewWeakPasswordDAOWithDB(db *gorm.DB) *WeakPasswordDAO {
	return &WeakPasswordDAO{db: db}
}

func (dao *WeakPasswordDAO) Exists(password string) (bool, error) {
	if dao.db == nil {
		return false, fmt.Errorf("weak-password database is not initialized")
	}

	var exists bool
	err := dao.db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM public.weak_password_dictionary
			WHERE LOWER(password) = ?
		)`, strings.ToLower(password)).Scan(&exists).Error
	if err != nil {
		return false, err
	}
	return exists, nil
}
