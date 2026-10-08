package dao

import (
	"alarm/internal/auth/dto"
	entity "alarm/models/model"
	"alarm/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserDAO struct {
	db *gorm.DB
}

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

	// 更新用户,所有非空字段更新
	if err := dao.db.Model(&userTem).Updates(&userTem).Error; err != nil {
		log.Printf("Fail to Update user: %v", err)
		return dbUser, false
	}
	return dbUser, true
}

func (dao *UserDAO) UpdateLogin(userTem entity.User) (entity.User, bool) {
	var dbUser entity.User
	// 更新用户token
	if err := dao.db.Model(&userTem).Updates(&userTem).Error; err != nil {
		// 更新失败
		log.Printf("Failed to update user login data: %v", err)
		return dbUser, false
	}
	return dbUser, true
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
