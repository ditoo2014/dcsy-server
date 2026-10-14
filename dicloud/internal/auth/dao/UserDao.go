package dao

import (
	"dicloud/internal/auth/dto"
	entity "dicloud/models/model"
	"dicloud/util/gormutil"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserDAO struct {
	db *gorm.DB
}

// ErrPasswordChanged 表示校验之后账号凭证已变化，调用方必须重新验证，不能覆盖并发改密结果。
var ErrPasswordChanged = errors.New("账号凭证已变化，请重新登录后重试")

// NewUserDAO 创建新的 UserDAO 实例
func NewUserDAO(ctx *gin.Context) *UserDAO {
	return &UserDAO{db: gormutil.GetGormDB(ctx, "")}
}

// GetAllUsers 获取所有用户
func (dao *UserDAO) GetAllUsers() ([]entity.User, error) {
	var users []entity.User

	// 使用 GORM 查询所有用户
	if err := dao.db.Table("users").Where("isadmin != 1 ").Find(&users).Error; err != nil {
		log.Printf("Failed to get users: %v", err)
		return nil, err
	}

	return users, nil
}

// CreateUser 创建新用户
func (dao *UserDAO) CreateUser(user entity.User) (int64, error) {
	// 使用 GORM 创建用户
	if err := dao.db.Create(&user).Error; err != nil {
		log.Printf("Failed to create user: %v", err)
		return 0, err
	}

	// 返回创建的用户 ID
	return int64(user.ID), nil
}

// 查询用户名是否已存在
func (dao *UserDAO) ExistId(userTem entity.User) (entity.User, bool) {
	var dbUser entity.User

	if err := dao.db.Table("users").Where("id = ?", userTem.ID).First(&dbUser).Error; err != nil {
		// 用户名不存在
		log.Printf("Failed to find user by id: %v", err)
		return dbUser, false
	}
	return dbUser, true
}

// 查询用户名是否已存在
func (dao *UserDAO) ExistName(userTem entity.User) (entity.User, bool) {
	var dbUser entity.User

	if err := dao.db.Table("users").Where("username = ?", userTem.Username).First(&dbUser).Error; err != nil {
		// 用户名不存在
		log.Printf("Failed to find user: %v", err)
		return dbUser, false
	}
	return dbUser, true
}

// 查询电话号码是否已存在
func (dao *UserDAO) ExistTelephone(userTem entity.User) (entity.User, bool) {
	var dbUser entity.User

	if err := dao.db.Table("users").Where("phone = ?", userTem.Phone).First(&dbUser).Error; err != nil {
		// 电话号码不存在
		log.Printf("Failed to find user: %v", err)
		return dbUser, false
	}
	return dbUser, true
}

// 查询用户名和token匹配的用户
func (dao *UserDAO) ExistNameAndToken(username string, token string) (entity.User, bool) {
	var dbUser entity.User

	if err := dao.db.Table("users").Where("username = ? AND token = ?", username, token).First(&dbUser).Error; err != nil {
		// 不存在能匹配的用户
		log.Printf("Failed to find user: %v", err)
		return dbUser, false
	}
	return dbUser, true
}

func (dao *UserDAO) Update(userTem entity.User) (entity.User, bool) {
	var dbUser entity.User
	if userTem.ID == 0 {
		return dbUser, false
	}
	// 普通资料更新采用明确白名单，不能改写密码、令牌、超管及密码安全状态。
	// 继续沿用原接口忽略零值的语义，避免未提交的资料被清空。
	profile := entity.User{
		TrueName: userTem.TrueName,
		Phone:    userTem.Phone,
		Email:    userTem.Email,
		ParentId: userTem.ParentId,
		PlatName: userTem.PlatName,
		PlatType: userTem.PlatType,
		Baned:    userTem.Baned,
		CorpIds:  userTem.CorpIds,
	}
	err := dao.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entity.User{}).Where("id = ?", userTem.ID).
			Updates(&profile)
		if result.Error != nil {
			return result.Error
		}
		return tx.Where("id = ?", userTem.ID).First(&dbUser).Error
	})
	if err != nil {
		log.Printf("Fail to Update user: %v", err)
		return dbUser, false
	}
	return dbUser, true
}

func (dao *UserDAO) UpdateLogin(userTem entity.User) (entity.User, bool) {
	var dbUser entity.User
	if userTem.ID == 0 || userTem.Password == "" || userTem.Token == "" {
		return dbUser, false
	}
	// 登录只能更新会话信息；旧密码条件防止验证后并发改密又写回旧令牌。
	err := dao.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entity.User{}).
			Where("id = ? AND password = ?", userTem.ID, userTem.Password).
			Updates(map[string]any{
				"token":       userTem.Token,
				"last_ip":     gorm.Expr("login_ip"),
				"login_ip":    userTem.LoginIp,
				"total_login": gorm.Expr("COALESCE(total_login, 0) + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrPasswordChanged
		}
		if err := tx.Where("id = ?", userTem.ID).First(&dbUser).Error; err != nil {
			return err
		}
		if dbUser.Token != userTem.Token {
			return ErrPasswordChanged
		}
		return nil
	})
	if err != nil {
		// 更新失败
		log.Printf("Failed to update user login data: %v", err)
		return dbUser, false
	}
	return dbUser, true
}

// ChangePassword 原子更新密码安全状态并失效数据库令牌。
// ID、旧密码和旧令牌必须仍匹配，防止并发登录、改密或管理员重置互相覆盖。
func (dao *UserDAO) ChangePassword(user entity.User, expectedHash string, changedAt time.Time, force bool) error {
	if user.ID == 0 || expectedHash == "" || user.Password == "" || changedAt.IsZero() {
		return fmt.Errorf("修改密码参数不完整")
	}
	return dao.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entity.User{}).
			Where("id = ? AND password = ? AND COALESCE(token, '') = ?", user.ID, expectedHash, user.Token).
			Updates(map[string]any{
				"password":             user.Password,
				"password_changed_at":  changedAt.UTC(),
				"must_change_password": force,
				"token":                "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrPasswordChanged
		}
		return nil
	})
}

func (dao *UserDAO) Delete(userTem entity.User) bool {
	if err := dao.db.Table("users").Delete(&userTem).Error; err != nil {
		log.Printf("Failed to delete user: %v", err)
		return false
	}
	return true
}

// 查询全部
func (dao *UserDAO) Find(u entity.User, queryStr string, offset, limit int) (_obj []dto.UserDto, total int64, err error) {

	/* 查询条件配置 */
	sqlStr := baseSqlFromat(u, queryStr)

	err = dao.db.Model(&u).Offset(offset).Limit(limit).Order("id").Where(sqlStr).Find(&_obj).Error
	dao.db.Model(&u).Where(sqlStr).Count(&total)
	if err != nil {
		log.Println("[UserDao] find fail", err)
		return
	}

	return
}

// GetUsersByIdArr 通过idarr获取用户
func (dao *UserDAO) GetUsersByIdArr(ids []int64) ([]entity.User, error) {
	var users []entity.User

	// 使用 GORM 查询用户
	if err := dao.db.Table("users").Where("isadmin != 1 ").Find(&users, ids).Error; err != nil {
		log.Printf("Failed to get users: %v", err)
		return nil, err
	}

	return users, nil
}

func (dao *UserDAO) FindByCorp(u entity.User, queryStr string, cpIds []int, offset, limit int) (_obj []dto.UserDto, total int64, err error) {

	/* 查询条件配置 */
	sqlStr := baseSqlFromat(u, queryStr)

	if len(cpIds) > 0 {
		// 根据部门查询
		idsStr := fmt.Sprintf("%d", cpIds[0])
		for i := 1; i < len(cpIds); i++ {
			idsStr = idsStr + fmt.Sprintf(",%d", cpIds[i])
		}
		sqlStr = sqlStr + " AND (users.corp_ids && array[" + idsStr + "]::int[] or users.corp_ids = '{}')"

	} else {
		// 只查询未分部门的 用户数据
		sqlStr = sqlStr + " AND users.corp_ids = '{}'"
	}

	err = dao.db.Model(&u).Offset(offset).Limit(limit).Order("id asc").Where(sqlStr).Find(&_obj).Error
	dao.db.Model(&u).Where(sqlStr).Count(&total)
	if err != nil {
		log.Println("[UserDao] find fail", err)
		return
	}

	return
}

// 指定确切部门查询, 是否真实存在绑定用户
func (dao *UserDAO) FindByExactCorpId(u entity.User, queryStr string, cpIds []int, offset, limit int) (_obj []dto.UserDto, total int64, err error) {

	/* 查询条件配置 */
	sqlStr := baseSqlFromat(u, queryStr)

	if len(cpIds) > 0 {
		// 根据部门查询
		idsStr := fmt.Sprintf("%d", cpIds[0])
		for i := 1; i < len(cpIds); i++ {
			idsStr = idsStr + fmt.Sprintf(",%d", cpIds[i])
		}
		sqlStr = sqlStr + " AND (users.corp_ids && array[" + idsStr + "]::int[])"

	} else {
		// 只查询未分部门的 用户数据
		sqlStr = sqlStr + " AND users.corp_ids = '{}'"
	}

	err = dao.db.Model(&u).Offset(offset).Limit(limit).Order("id asc").Where(sqlStr).Find(&_obj).Error
	dao.db.Model(&u).Where(sqlStr).Count(&total)
	if err != nil {
		log.Println("[UserDao] find fail", err)
		return
	}

	return
}

// 封装基本查询SQL
func baseSqlFromat(u entity.User, queryStr string) (sqlStr string) {

	sqlStr = " isadmin != 1 "
	if u.ID > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND id = %d ", u.ID)
	}
	if len(u.TrueName) > 0 {
		sqlStr = sqlStr + " AND true_name LIKE '%" + u.TrueName + "%' "
	}
	if len(u.Username) > 0 {
		sqlStr = sqlStr + " AND username LIKE '%" + u.Username + "%' "
	}
	if len(u.Phone) > 0 {
		sqlStr = sqlStr + " AND phone LIKE '%" + u.Phone + "%' "
	}
	if len(queryStr) > 0 {
		sqlStr = sqlStr + " AND ( true_name LIKE '%" + queryStr + "%' " +
			" or username LIKE '%" + queryStr + "%' " +
			" or phone LIKE '%" + queryStr + "%' )"
	}

	return
}
