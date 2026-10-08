package controller

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"dicloud/internal/auth/dao"
	"dicloud/internal/auth/dto"
	"dicloud/internal/common/service"
	"dicloud/models/model"
	"dicloud/pkg/authsession"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/util/encryption"
	requestutil "dicloud/util/http"
	normalutil "dicloud/util/normal"
	"dicloud/util/response"

	"github.com/spf13/viper"
)

type IUserCtrler interface {
	Register(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Info(ctx *gin.Context)
	Login(ctx *gin.Context)
	Logout(ctx *gin.Context)
	SendSmsCode(ctx *gin.Context)
	FindPage(ctx *gin.Context)

	ChangePassword(ctx *gin.Context)
	ResetUserPassword(ctx *gin.Context)
}

type UserCtrler struct {
}

// Logout immediately invalidates the current Redis login session.
// Missing or already-expired sessions are treated as a successful logout.
func (u UserCtrler) Logout(ctx *gin.Context) {
	authorization := strings.TrimSpace(ctx.GetHeader("authorization"))
	token := ""
	if strings.HasPrefix(authorization, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	}
	if err := authsession.Clear(ctx, token); err != nil {
		log.Println("[AUTH] Logout ClearSession", err)
		response.Fail(ctx, "退出登录失败", nil)
		return
	}
	response.Success(ctx, "退出登录成功", nil)
}

const SMS_VERIFYCODE_PREFIX = "smsverifycode:%s"
const SMS_VERIFYCODE_LIMIT_PREFIX = "smsverifylimit:%s"
const SMS_SEND_URL_DEFAULT = "https://security.sctower.cn/gapi/open-api/sms/sendmessage"

type loginSmsCache struct {
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Code     string `json:"code"`
}

type smsSendResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// 用户注册
func (u UserCtrler) Register(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 获取信息 */

	var rUser model.User
	ctx.Bind(&rUser)

	/* RSA PKCS8解密 */
	if len(rUser.Password) < 1 {
		response.Fail(ctx, "无效密码", nil)
		return
	}

	pwd, err := encryption.DefaultPKCS8Decrypt(rUser.Password)
	if err != nil {
		log.Println("[AUTH] Register RSA Decrypt Fail: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}
	rUser.Password = pwd

	/* 验证信息格式 */
	if len(rUser.TrueName) < 1 {
		response.Fail(ctx, "用户姓名为空", nil)
		return
	}

	if len(rUser.Username) < 1 {
		response.Fail(ctx, "账号不能为空", nil)
		return
	}

	if len(rUser.Phone) != 11 {
		response.Fail(ctx, "手机号必须为11位", nil)
		return
	}

	// 多租户, 区分租户操作
	if len(rUser.Renter) == 0 {
		rUser.Renter = userInfo.Renter
	}

	if err := validateSubmittedPassword(rUser.Password, rUser); err != nil {
		respondPasswordPolicyError(ctx, err)
		return
	}

	/* 数据合法性判断 */

	match, _ := regexp.MatchString(model.USERNAME_PATTERN, rUser.Username)
	if !match {
		response.Fail(ctx, "账号不合法", nil)
		return
	}
	match, _ = regexp.MatchString(model.PHONT_PATTERN, rUser.Phone)
	if !match {
		response.Fail(ctx, "电话号码不合法", nil)
		return
	}

	/* 初始化dao */
	userDao := dao.NewUserDAO(ctx)

	/* 验证用户名 */

	_, res := userDao.ExistName(rUser)
	if res {
		response.Fail(ctx, "账户名称已被注册", nil)
		return
	}

	/* 验证号码 */

	_, res = userDao.ExistTelephone(rUser)
	if res {
		response.Fail(ctx, "账户电话号码已被注册", nil)
		return
	}

	/* 加密密码 */

	hasedPassword, err := bcrypt.GenerateFromPassword([]byte(rUser.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("[AUTH] Register 密码加密错误: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}

	rUser.Password = string(hasedPassword)

	/* 设置权限级别 */
	rUser.Isadmin = 0
	rUser.Power = 2
	if len(rUser.CorpIds) == 0 {
		rUser.CorpIds = append(rUser.CorpIds, 1)
	}

	/* 信息入库 */
	rUser.CreateTime = time.Now()
	_, err = userDao.CreateUser(rUser)
	if nil != err {
		log.Println("[AUTH] Register Create: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}

	/* 返回成功 */
	ctx.JSON(200, gin.H{"code": 200, "msg": "注册成功"})

}

// 修改用户信息
func (u UserCtrler) Update(ctx *gin.Context) {

	/* 获取信息 */

	var rUser model.User
	ctx.Bind(&rUser)

	if rUser.ID == 0 {
		response.Fail(ctx, "非法数据", nil)
		return
	}

	/* 信息入库 */
	userDao := dao.NewUserDAO(ctx)
	rUser.Isadmin = 0   // 禁止修改用户超管权限
	rUser.Username = "" // 禁止修改用户名
	rUser.Password = "" // 禁止修改密码, 调用重置密码接口

	rUser, res := userDao.Update(rUser)
	if !res {
		log.Println("[AUTH] Update Fail")
		response.Fail(ctx, "系统错误", nil)
		return
	}

	/* 返回成功 */
	cache.HMSet(ctx, fmt.Sprintf("user:%s", rUser.Username), map[string]interface{}{
		"info": rUser,
	})

	response.Success(ctx, "修改成功", nil)
}

// 删除用户
func (u UserCtrler) Delete(ctx *gin.Context) {

	/* 获取信息 */

	var rUser model.User
	ctx.Bind(&rUser)

	/* 验证账号 */

	var res bool
	userDao := dao.NewUserDAO(ctx)

	if rUser.ID == 0 {
		response.Fail(ctx, "非法数据", nil)
		return
	}

	// 查询用户是否绑定角色
	userRoleDao := dao.NewUserRoleDAO(ctx)
	dataTem, res := userRoleDao.ExistRoleBindUser(&rUser)
	if res && dataTem.ID > 0 {
		response.Fail(ctx, "请先将用户角色解绑再删除", nil)
		return
	}

	/* 信息出库 */

	res = userDao.Delete(rUser)
	if !res {
		log.Println("[AUTH] Delete Fail")
		response.Fail(ctx, "系统错误", nil)
		return
	}

	/* 返回成功 */

	response.Success(ctx, "删除成功", nil)
}

// 查询登录用户信息
func (u UserCtrler) Info(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 返回成功 */

	response.Success(
		ctx,
		"认证成功",
		gin.H{"user": dto.UserDto{
			ID:         userInfo.ID,
			Username:   userInfo.Username,
			TrueName:   userInfo.TrueName,
			Phone:      userInfo.Phone,
			Email:      userInfo.Email,
			CreateTime: userInfo.CreateTime,
			TotalLogin: userInfo.TotalLogin,
			Token:      userInfo.Token,
			LoginIp:    userInfo.LastIp,
			LastIp:     userInfo.LastIp,
			Isadmin:    userInfo.Isadmin,
			PlatName:   userInfo.PlatName,
			PlatType:   userInfo.PlatType,
			Baned:      userInfo.Baned,
			CorpIds:    userInfo.CorpIds,
		}})
}

// 查询用户
func (u UserCtrler) FindPage(ctx *gin.Context) {

	var (
		pageStr     string
		pageSizeStr string
		corpIdStr   string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	var queryStr = ctx.Query("query")

	/* 获取信息 */

	var userCofing model.User
	ctx.Bind(&userCofing)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	// fmt.Println("pageNum = ", pageStr, ", pageSize = ", pageSizeStr)
	pageNum, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(pageNum, pageSize)

	corpIdStr = ctx.Query("corpId")
	corpId, _ := strconv.Atoi(corpIdStr)

	/* 查询 */

	var resMsg gin.H

	userDao := dao.NewUserDAO(ctx)

	if userInfo.Isadmin == 1 {

		// 部门筛选
		if corpId > 0 {
			var corpIds []int
			// 单独部门操作
			corpIds = append(corpIds, int(corpId))
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			data, total, err := userDao.FindByCorp(userCofing, queryStr, corpIds, offset, limit)
			if err != nil {
				response.Fail(ctx, "查询失败", nil)
				return
			}
			resMsg = gin.H{"data": data, "total": total}

		} else {

			data, total, err := userDao.Find(userCofing, queryStr, offset, limit)
			if err != nil {
				response.Fail(ctx, "查询失败", nil)
				return
			}

			resMsg = gin.H{"data": data, "total": total}
		}
	} else {

		// 根据部门查询

		// 查询用户 全部部门
		// 查询全部子部门
		corpIds := []int{}
		for _, v := range userInfo.CorpIds {
			// 加上根节点
			corpIds = append(corpIds, int(v))
		}
		// 子部门查询
		corpIds = service.FindAllSonCorp(ctx, corpIds)
		// 查询部门用户id

		// 鉴权部门
		if corpId > 0 && len(corpIds) > 0 {
			idExist := -1
			for _, ID := range corpIds {
				if ID == corpId {
					idExist = 1
					break
				}
			}
			if idExist == -1 {
				log.Println("[AUTH] not exist corp: ", corpId)
				response.Fail(ctx, "非法部门, 未授权", nil)
				return
			}

			// 重新筛选
			corpIds = []int{corpId}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)
		}

		// fmt.Println("pageSize = " + pageSizeStr + ", pageNum = " + pageStr)
		data, total, err := userDao.FindByCorp(userCofing, queryStr, corpIds, offset, limit)
		if err != nil {
			response.Fail(ctx, "查询失败", nil)
			return
		}

		resMsg = gin.H{"data": data, "total": total}
	}

	/* 返回成功 */
	response.Success(ctx, "success", resMsg)
}

// 登录
func (u UserCtrler) Login(ctx *gin.Context) {

	/* 获取信息 */

	var rDto dto.LoginDTO
	if err := ctx.ShouldBind(&rDto); err != nil {
		log.Println("[AUTH] Login Bind Fail: ", err)
		response.Fail(ctx, "用户名或密码错误", nil)
		return
	}

	rUser := model.User{
		Username: strings.TrimSpace(rDto.Username),
		Password: rDto.Password,
		Phone:    strings.TrimSpace(rDto.Phone),
		PlatId:   rDto.PlatId,
	}
	loginIdentifier := loginIdentifierFromDTO(rDto)

	if len(loginIdentifier) == 0 || len(rUser.Password) == 0 {
		log.Println("[AUTH] Login 用户名或密码为空")
		response.Fail(ctx, "用户名或密码错误", nil)
		return
	}
	if !ensureLoginAllowed(ctx, loginIdentifier) {
		return
	}

	/* RSA PKCS8解密 */

	pwd, err := encryption.DefaultPKCS8Decrypt(rUser.Password)
	if err != nil {
		log.Println("[AUTH] Login RSA Decrypt Fail: ", err)
		failLogin(ctx, loginIdentifier)
		return
	}
	rUser.Password = pwd

	/* 验证信息 */

	var loginObject string

	if 0 < len(rUser.Username) {
		loginObject = "name"
	} else if len(rUser.Phone) == 11 {
		loginObject = "telephone"
	} else {
		log.Println("[AUTH] Login 名字为空或号码长度不为11")
		failLogin(ctx, loginIdentifier)
		return
	}

	/* 验证账号 */

	var dbUser model.User
	var res bool
	userDao := dao.NewUserDAO(ctx)

	switch loginObject {
	case "name":
		dbUser, res = userDao.ExistName(rUser)
		if !res {
			log.Println("[AUTH] Login 账户名称不存在")
			failLogin(ctx, loginIdentifier)
			return
		}
	case "telephone":
		dbUser, res = userDao.ExistTelephone(rUser)
		if !res {
			log.Println("[AUTH] Login 账户号码不存在")
			failLogin(ctx, loginIdentifier)
			return
		}
	}
	accountIdentifier := loginIdentifierFromUser(dbUser, loginIdentifier)
	if accountIdentifier != loginIdentifier && !ensureLoginAllowed(ctx, accountIdentifier) {
		return
	}

	/* 验证密码 */

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(rUser.Password)); err != nil {
		log.Println("[AUTH] Login 密码错误")
		failLogin(ctx, accountIdentifier)
		return
	}

	if !verifyLoginSmsCode(ctx, rDto, dbUser) {
		return
	}

	/* 生成token */

	token, timeDuration, err := common.ReleaseToken(dbUser.Username)
	if err != nil {
		log.Println("[AUTH] Login ReleaseToken", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}

	/* 更新token */

	dbUser.Token = token
	dbUser.LastIp = dbUser.LoginIp
	dbUser.LoginIp = ctx.RemoteIP()
	dbUser.TotalLogin = dbUser.TotalLogin + 1
	userDao.UpdateLogin(dbUser)

	/* 更新redis */

	dbUser.Password = ""
	ctx.Set("userInfo", dbUser)
	cache.HMSet(ctx, fmt.Sprintf("user:%s", dbUser.Username), map[string]interface{}{
		"info": dbUser,
	})
	// 设置过期时间
	cache.ExpireKey(ctx, fmt.Sprintf("user:%s", dbUser.Username), timeDuration)
	if err := authsession.Activate(ctx, dbUser.Username, token); err != nil {
		log.Println("[AUTH] Login ActivateSession", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}

	// 权限查询
	permissionDao := dao.NewPermissionDAO(ctx)
	pmss, err := permissionDao.GetUserPermissions(dbUser, rUser.PlatId)
	if err == nil {
		cache.HMSet(ctx, fmt.Sprintf("user:%s", dbUser.Username), map[string]interface{}{
			fmt.Sprintf("permission%v", rUser.PlatId): pmss,
		})
		// 设置过期时间
		cache.ExpireKey(ctx, fmt.Sprintf("user:%s", dbUser.Username), timeDuration)
	} else {
		cache.ExpireKey(ctx, fmt.Sprintf("user:%s", dbUser.Username), time.Second*3)
	}
	clearLoginFailure(ctx, loginIdentifier)
	if accountIdentifier != loginIdentifier {
		clearLoginFailure(ctx, accountIdentifier)
	}

	/* 返回成功 */

	response.Success(ctx, "登录成功", gin.H{"token": token})
}

// 发送登录短信验证码
func (u UserCtrler) SendSmsCode(ctx *gin.Context) {
	var rDto dto.LoginDTO
	if err := ctx.ShouldBind(&rDto); err != nil {
		log.Println("[AUTH] SendSmsCode Bind Fail: ", err)
		response.Fail(ctx, "用户名或密码错误", nil)
		return
	}
	rDto.Username = strings.TrimSpace(rDto.Username)
	rDto.Phone = strings.TrimSpace(rDto.Phone)
	loginIdentifier := loginIdentifierFromDTO(rDto)

	if len(loginIdentifier) == 0 || len(rDto.Password) == 0 {
		response.Fail(ctx, "用户名或密码错误", nil)
		return
	}
	if !ensureLoginAllowed(ctx, loginIdentifier) {
		return
	}

	userDao := dao.NewUserDAO(ctx)
	dbUser, res := userDao.ExistName(model.User{Username: rDto.Username})
	if !res {
		log.Println("[AUTH] SendSmsCode 账户名称不存在")
		failLogin(ctx, loginIdentifier)
		return
	}
	accountIdentifier := loginIdentifierFromUser(dbUser, loginIdentifier)
	if accountIdentifier != loginIdentifier && !ensureLoginAllowed(ctx, accountIdentifier) {
		return
	}

	pwd, err := encryption.DefaultPKCS8Decrypt(rDto.Password)
	if err != nil {
		log.Println("[AUTH] SendSmsCode RSA Decrypt Fail: ", err)
		failLogin(ctx, accountIdentifier)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(pwd)); err != nil {
		log.Println("[AUTH] SendSmsCode 密码错误")
		failLogin(ctx, accountIdentifier)
		return
	}

	if !validPhone(dbUser.Phone) {
		response.Fail(ctx, "当前账号未绑定有效手机号", nil)
		return
	}

	limitSeconds := viper.GetInt("sms.login.intervalSeconds")
	if limitSeconds <= 0 {
		limitSeconds = 60
	}
	limitKey := fmt.Sprintf(SMS_VERIFYCODE_LIMIT_PREFIX, dbUser.Username)
	if result, err := cache.Get(ctx, limitKey); err == nil && len(result) > 0 {
		response.Fail(ctx, "验证码发送过于频繁，请稍后再试", nil)
		return
	}

	uid, err := uuid.NewUUID()
	if err != nil {
		response.Fail(ctx, "获取验证码失败", nil)
		return
	}
	uidStr := strings.ReplaceAll(uid.String(), "-", "")

	// 模拟模式仍然生成并保存短信验证码，只跳过外部短信接口，保证前端和登录校验流程不变。
	code := ""
	if viper.GetBool("sms.login.simulate") {
		code = strings.TrimSpace(viper.GetString("sms.login.simulateCode"))
		validMockCode, _ := regexp.MatchString(`^\d{6}$`, code)
		if !validMockCode {
			log.Println("[AUTH] SendSmsCode 模拟验证码配置无效，要求六位数字")
			response.Fail(ctx, "模拟验证码配置无效", nil)
			return
		}
		log.Println("[AUTH] SendSmsCode 已启用模拟模式，未调用外部短信接口")
	} else {
		var err error
		code, err = getRandomDigitCode(6)
		if err != nil {
			log.Println("[AUTH] SendSmsCode Random Code Fail: ", err)
			response.Fail(ctx, "获取验证码失败", nil)
			return
		}

		if err := sendLoginSms(dbUser.Phone, code); err != nil {
			log.Println("[AUTH] SendSmsCode Send Fail: ", err)
			response.Fail(ctx, fmt.Sprintf("短信发送失败：%s", err.Error()), nil)
			return
		}
	}

	expireSeconds := viper.GetInt("sms.login.expireSeconds")
	if expireSeconds <= 0 {
		expireSeconds = 300
	}
	cacheData := loginSmsCache{
		Username: dbUser.Username,
		Phone:    dbUser.Phone,
		Code:     code,
	}
	cacheBytes, err := json.Marshal(cacheData)
	if err != nil {
		log.Println("[AUTH] SendSmsCode Marshal Fail: ", err)
		response.Fail(ctx, "获取验证码失败", nil)
		return
	}

	key := fmt.Sprintf(SMS_VERIFYCODE_PREFIX, uidStr)
	if err := cache.Set(ctx, key, string(cacheBytes)); err != nil {
		log.Println("[AUTH] SendSmsCode Cache Set Fail: ", err)
		response.Fail(ctx, "获取验证码失败", nil)
		return
	}
	cache.ExpireKey(ctx, key, time.Duration(expireSeconds)*time.Second)
	if err := cache.Set(ctx, limitKey, "1"); err != nil {
		log.Println("[AUTH] SendSmsCode Limit Cache Set Fail: ", err)
	}
	cache.ExpireKey(ctx, limitKey, time.Duration(limitSeconds)*time.Second)

	response.Success(ctx, "短信验证码已发送", gin.H{
		"smsCodeId":       uidStr,
		"phone":           maskPhone(dbUser.Phone),
		"expireSeconds":   expireSeconds,
		"intervalSeconds": limitSeconds,
	})
}

func getRandomDigitCode(length int) (string, error) {
	var builder strings.Builder
	for i := 0; i < length; i++ {
		num, err := crand.Int(crand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		builder.WriteString(num.String())
	}

	return builder.String(), nil
}

func sendLoginSms(phone string, code string) error {
	smsURL := viper.GetString("sms.login.url")
	if len(smsURL) == 0 {
		smsURL = SMS_SEND_URL_DEFAULT
	}

	message := fmt.Sprintf("您的登录验证码为：%s，5分钟内有效，请勿泄露。", code)
	if template := viper.GetString("sms.login.messageTemplate"); len(template) > 0 {
		if strings.Contains(template, "%s") {
			message = fmt.Sprintf(template, code)
		} else if strings.Contains(template, "{code}") {
			message = strings.ReplaceAll(template, "{code}", code)
		} else {
			message = template
		}
	}

	timeoutSeconds := viper.GetInt("sms.login.timeoutSeconds")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 20
	}

	respBody, err := requestutil.PostMultipartFormWithTimeout(smsURL, map[string]string{
		"phone":   phone,
		"message": message,
	}, time.Duration(timeoutSeconds)*time.Second)
	if err != nil {
		return err
	}

	var smsResp smsSendResponse
	if err := json.Unmarshal(respBody, &smsResp); err != nil {
		return fmt.Errorf("parse sms response fail: %w, body: %s", err, string(respBody))
	}
	if smsResp.Code != 200 {
		return fmt.Errorf("sms response fail, code: %d, msg: %s", smsResp.Code, smsResp.Msg)
	}

	return nil
}

func verifyLoginSmsCode(ctx *gin.Context, req dto.LoginDTO, dbUser model.User) bool {
	smsCodeId := strings.TrimSpace(req.SmsCodeId)
	smsCode := strings.TrimSpace(req.SmsCode)
	if len(smsCodeId) == 0 || len(smsCode) == 0 {
		response.Fail(ctx, "请输入短信验证码", nil)
		return false
	}

	key := fmt.Sprintf(SMS_VERIFYCODE_PREFIX, smsCodeId)
	result, err := cache.Get(ctx, key)
	if err != nil || len(result) == 0 {
		failLoginWithMessage(ctx, dbUser.Username, "短信验证码错误或已过期")
		return false
	}

	var cacheData loginSmsCache
	if err := json.Unmarshal([]byte(result), &cacheData); err != nil {
		log.Println("[AUTH] verifyLoginSmsCode Unmarshal Fail: ", err)
		failLoginWithMessage(ctx, dbUser.Username, "短信验证码错误或已过期")
		return false
	}

	if cacheData.Code != smsCode || cacheData.Username != dbUser.Username || cacheData.Phone != dbUser.Phone {
		failLoginWithMessage(ctx, dbUser.Username, "短信验证码错误或已过期")
		return false
	}

	cache.Del(ctx, key)
	return true
}

func validPhone(phone string) bool {
	if len(phone) != 11 {
		return false
	}
	match, _ := regexp.MatchString(model.PHONT_PATTERN, phone)
	return match
}

func maskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// 修改密码
func (u UserCtrler) ChangePassword(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 获取信息 */

	userDao := dao.NewUserDAO(ctx)
	var rDto dto.UserNewPasswordDTO
	ctx.Bind(&rDto)

	if len(rDto.Password) == 0 || len(rDto.NewPassword) == 0 {
		response.Fail(ctx, "非法数据", nil)
		return
	}

	/* RSA PKCS8解密 */
	/* 登录用户密码验证 */

	pwd, err := encryption.DefaultPKCS8Decrypt(rDto.Password)
	if err != nil {
		log.Println("[AUTH] ChangePassword RSA Decrypt Fail: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}
	rDto.Password = pwd
	userInfo, res := userDao.ExistId(userInfo)
	if !res {
		log.Println("[AUTH] ChangePassword login user find Fail: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}
	// 验证
	if err := bcrypt.CompareHashAndPassword([]byte(userInfo.Password), []byte(rDto.Password)); err != nil {
		log.Println("[AUTH] ChangePassword 密码错误 Fail: ", err)
		response.Fail(ctx, "密码错误", nil)
		return
	}

	/* 新密码解密 */

	newPwd, err := encryption.DefaultPKCS8Decrypt(rDto.NewPassword)
	if err != nil {
		log.Println("[AUTH] ChangePassword RSA Decrypt New Fail: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}
	// 校验
	if err := validateSubmittedPassword(newPwd, userInfo); err != nil {
		log.Println("[AUTH] ChangePassword 密码策略校验失败: ", err)
		respondPasswordPolicyError(ctx, err)
		return
	}

	/* 加密密码 */

	hasedPassword, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		log.Println("[AUTH] ChangePassword 密码加密错误")
		response.Fail(ctx, "系统错误", nil)
		return
	}
	rDto.NewPassword = string(hasedPassword)

	/* 信息入库 */

	userInfo.Password = rDto.NewPassword

	rUser, res := userDao.Update(userInfo)
	if !res {
		log.Println("[AUTH] ChangePassword Fail")
		response.Fail(ctx, "系统错误", nil)
		return
	}

	/* 返回成功 */
	cache.HMSet(ctx, fmt.Sprintf("user:%s", rUser.Username), map[string]interface{}{
		"info": rUser,
	})

	response.Success(ctx, "修改成功", nil)

}

// 管理员重置用户密码
func (u UserCtrler) ResetUserPassword(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 获取信息 */

	userDao := dao.NewUserDAO(ctx)
	var rDto dto.ResetUserPwdDTO
	ctx.Bind(&rDto)

	if rDto.UsersEntity.ID == 0 || len(rDto.ManagerPassword) == 0 || len(rDto.NewPassword) == 0 {
		response.Fail(ctx, "非法数据", nil)
		return
	}

	/* RSA PKCS8解密 */

	/* 登录用户密码验证 */
	pwd, err := encryption.DefaultPKCS8Decrypt(rDto.ManagerPassword)
	if err != nil {
		log.Println("[AUTH] ResetUserPassword RSA Decrypt Fail: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}
	rDto.ManagerPassword = pwd
	userInfo, res := userDao.ExistId(userInfo)
	if !res {
		log.Println("[AUTH] ChangePassword login user find Fail: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}
	// 验证
	if err := bcrypt.CompareHashAndPassword([]byte(userInfo.Password), []byte(rDto.ManagerPassword)); err != nil {
		log.Println("[AUTH] ResetUserPassword 密码错误 Fail: ", err)
		response.Fail(ctx, "密码错误", nil)
		return
	}

	/* 新密码解密 */

	newPwd, err := encryption.DefaultPKCS8Decrypt(rDto.NewPassword)
	if err != nil {
		log.Println("[AUTH] ResetUserPassword RSA Decrypt New Fail: ", err)
		response.Fail(ctx, "系统错误", nil)
		return
	}

	// 查询用户
	dbUser, res := userDao.ExistId(rDto.UsersEntity)
	if !res {
		response.Fail(ctx, "无效用户", nil)
		return
	}

	if err := validateSubmittedPassword(newPwd, dbUser); err != nil {
		log.Println("[AUTH] ResetUserPassword 密码策略校验失败: ", err)
		respondPasswordPolicyError(ctx, err)
		return
	}

	/* 加密密码 */

	hasedPassword, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		log.Println("[AUTH] ResetUserPassword 密码加密错误")
		response.Fail(ctx, "系统错误", nil)
		return
	}
	rDto.NewPassword = string(hasedPassword)

	/* 信息入库 */

	// 更新密码
	dbUser.Password = rDto.NewPassword
	dbUser, res = userDao.Update(dbUser)
	if !res {
		log.Println("[AUTH] ResetUserPassword Fail")
		response.Fail(ctx, "系统错误", nil)
		return
	}

	/* 返回成功 */
	cache.HMSet(ctx, fmt.Sprintf("user:%s", dbUser.Username), map[string]interface{}{
		"info": dbUser,
	})

	response.Success(ctx, "修改成功", nil)
}
