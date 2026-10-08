package service

import (
	"alarm/internal/alarm/dao"
	"alarm/internal/alarm/model"
	commonmodel "alarm/internal/common/model"
	"alarm/internal/common/service"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

type SmsTem struct {
	SignName            string
	Endpoint            string
	RegionId            string
	CurClient           *dysmsapi20170525.Client
	AlarmTemplateCode   string
	AlarmSite           string
	AlarmEquipment      string
	AlarmCoding         string
	AlarmValue          string
	RecoverTemplateCode string
	RecoverSite         string
	RecoverEquipment    string
	RecoverCoding       string
	RecoverValue        string
}

var (
	// 部门分层客户端, 通过corpId区分不同客户
	// 租户分层客户端, 通过租户名称区分不同数据库, 获取不同客户端

	// 租户的客户端 -> 租户: 部门: 客户端
	smsRenterClients = make(map[string]map[string]*SmsTem)
	smsClientsMu     sync.RWMutex
)

func setSmsClients(scope string, clients map[string]*SmsTem) {
	smsClientsMu.Lock()
	smsRenterClients[scope] = clients
	smsClientsMu.Unlock()
}

func getSmsClients(scope string) map[string]*SmsTem {
	smsClientsMu.RLock()
	clients := smsRenterClients[scope]
	smsClientsMu.RUnlock()
	return clients
}

func getSmsScopeKey(ctx *gin.Context) string {
	dbName, exists := ctx.Get("dbName")
	if !exists {
		return ""
	}
	return fmt.Sprint(dbName)
}

// 初始化租户部门客户端, 通过数据库数据初始化; 请先初始化数据库链接, 并指定数据库
func InitAllSmsClient(ctx *gin.Context) error {

	smsDao := dao.NewSmsDao(ctx)
	smsses := smsDao.GetAllSecret(model.SmsSecret{})
	// 初始化所有秘钥信息
	var corpSms = map[string]*SmsTem{}
	if len(smsses) > 0 {
		for _, smss := range smsses {
			// 部门秘钥信息
			if smss.ID == 0 {
				log.Println("[ALARM] ali sms illegal data")
				continue
			}

			smsTem, err := createSmsClient(&smss)
			if err != nil {
				log.Printf("[ALARM] ali sms create client fail, credentialId=%d", smss.ID)
				continue
			}

			// 基本参数填充
			// 非空项 - 签名, 报警模板, 恢复模板
			smsTem.SignName = smss.SignName
			smsTem.AlarmTemplateCode = smss.AlarmTemplateCode
			smsTem.RecoverTemplateCode = smss.RecoverTemplateCode

			// 服务地址, 区域ID
			if len(smss.Endpoint) == 0 {
				smsTem.Endpoint = model.SMS_ENDPOINT // 默认服务地址
			} else {
				smsTem.Endpoint = smss.Endpoint
			}
			if len(smss.RegionId) == 0 {
				smsTem.RegionId = model.SMS_REGIONID_DEFAULT // 默认区域ID
			} else {
				smsTem.RegionId = smss.RegionId
			}
			// 报警模板参数 - 以防申请时候不一致
			if len(smss.AlarmSite) == 0 {
				smsTem.AlarmSite = model.SMS_ALARM_SITE_TEMPLATE // 默认参数名
			} else {
				smsTem.AlarmSite = smss.AlarmSite
			}
			if len(smss.AlarmEquipment) == 0 {
				smsTem.AlarmEquipment = model.SMS_ALARM_EQUIPMENT_TEMPLATE // 默认参数名
			} else {
				smsTem.AlarmEquipment = smss.AlarmEquipment
			}
			if len(smss.AlarmCoding) == 0 {
				smsTem.AlarmCoding = model.SMS_ALARM_CODING_TEMPLATE // 默认参数名
			} else {
				smsTem.AlarmCoding = smss.AlarmCoding
			}
			if len(smss.AlarmValue) == 0 {
				smsTem.AlarmValue = model.SMS_ALARM_VALUE_TEMPLATE // 默认参数名
			} else {
				smsTem.AlarmValue = smss.AlarmValue
			}

			// 恢复报警模板参数 - 以防申请时候不一致
			if len(smss.RecoverSite) == 0 {
				smsTem.RecoverSite = model.SMS_RECOVER_SITE_TEMPLATE // 默认参数名
			} else {
				smsTem.RecoverSite = smss.RecoverSite
			}
			if len(smss.RecoverEquipment) == 0 {
				smsTem.RecoverEquipment = model.SMS_RECOVER_EQUIPMENT_TEMPLATE // 默认参数名
			} else {
				smsTem.RecoverEquipment = smss.RecoverEquipment
			}
			if len(smss.RecoverCoding) == 0 {
				smsTem.RecoverCoding = model.SMS_RECOVER_CODING_TEMPLATE // 默认参数名
			} else {
				smsTem.RecoverCoding = smss.RecoverCoding
			}
			if len(smss.RecoverValue) == 0 {
				smsTem.RecoverValue = model.SMS_RECOVER_VALUE_TEMPLATE // 默认参数名
			} else {
				smsTem.RecoverValue = smss.RecoverValue
			}

			// 部门client - 如果部门id为0, 则是默认客户端, 谁都可以使用
			corpSms[fmt.Sprint(smss.CorpId)] = smsTem
		}
	} else {
		log.Println("[ALARM] ali sms 未查询到秘钥数据")
		setSmsClients(getSmsScopeKey(ctx), corpSms)
		return nil
	}

	setSmsClients(getSmsScopeKey(ctx), corpSms)

	return nil
}

// 初始化客户端数据
func createSmsClient(smss *model.SmsSecret) (smsTem *SmsTem, _err error) {

	// 工程代码泄露可能会导致 AccessKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考。
	// 建议使用更安全的 STS 方式，更多鉴权访问方式请参见：https://help.aliyun.com/document_detail/378661.html。
	config := &openapi.Config{
		// 必填，请确保代码运行环境设置了环境变量 ALIBABA_CLOUD_ACCESS_KEY_ID。
		AccessKeyId: tea.String(smss.AkId),
		// 必填，请确保代码运行环境设置了环境变量 ALIBABA_CLOUD_ACCESS_KEY_SECRET。
		AccessKeySecret: tea.String(smss.AkSecret),
	}

	// Endpoint 请参考 https://api.aliyun.com/product/Dysmsapi
	config.Endpoint = tea.String(smss.Endpoint)
	config.RegionId = tea.String(smss.RegionId)

	smsTem = &SmsTem{}
	smsTem.CurClient, _err = dysmsapi20170525.NewClient(config)
	if _err != nil {
		fmt.Println("[ALARM] ali sms 初始化客户端失败 ", _err)
		return nil, _err
	}

	return
}

// 阿里云短信消息
func AliSendSms(ctx *gin.Context, data *commonmodel.AlarmMQModel) {

	// 封装消息时, 加锁; 防止客户端和消息对不上
	if data == nil || len(data.AlarmUsers) == 0 {
		log.Println("[ALARM] send alimsg service. user is null")
		return
	}

	// 报警
	if data.End <= 1 {
		sendSmsAlarm(ctx, data)
	} else {
		// 恢复
		sendSmsRecover(ctx, data)
	}
}

// 发送短信报警
func sendSmsAlarm(ctx *gin.Context, a *commonmodel.AlarmMQModel) (_err error) {

	// 获取报警消息对应的, 客户端
	smsTem := getSmsTem(ctx, a)
	if smsTem == nil || smsTem.CurClient == nil {
		log.Println("[ALARM] send alarm fail, client not exist")
		return
	}

	// 报警用户填充, 分批 -> 批量发送
	for _, smsReq := range getAlarmSmsRequest(ctx, a) {

		// 其他数据填充
		smsReq.SetSignName(smsTem.SignName)

		// 报警模板: SMS_218727965
		// 报警恢复模板: SMS_218727963
		smsReq.SetTemplateCode(smsTem.AlarmTemplateCode)
		smsReq.SetTemplateParam("{\"" + smsTem.AlarmSite + "\":\"" + a.SiteName + "\",\"" +
			smsTem.AlarmEquipment + "\":\"" + a.EquipmentName + "\",\"" +
			smsTem.AlarmCoding + "\":\"" + a.AlarmCode + "\",\"" +
			smsTem.AlarmValue + "\":\"" + fmt.Sprint(a.AlarmValue) + "\"}")

		// 发送并接受返回结果
		tryErr := func() (_e error) {
			defer func() {
				if r := tea.Recover(recover()); r != nil {
					_e = r
				}
			}()

			// 复制代码运行请自行打印 API 的返回值
			_, _err = smsTem.CurClient.SendSmsWithOptions(smsReq, &util.RuntimeOptions{})
			if _err != nil {
				return _err
			}
			return nil
		}()

		// TODO: 问题处理
		if tryErr != nil {
			var errorTem = &tea.SDKError{}
			if _t, ok := tryErr.(*tea.SDKError); ok {
				errorTem = _t
			} else {
				errorTem.Message = tea.String(tryErr.Error())
			}
			// 此处仅做打印展示，请谨慎对待异常处理，在工程项目中切勿直接忽略异常。
			// 错误 message
			log.Println("[SMS] error: ", tea.StringValue(errorTem.Message))
			// 诊断地址
			var data interface{}
			d := json.NewDecoder(strings.NewReader(tea.StringValue(errorTem.Data)))
			d.Decode(&data)
			if m, ok := data.(map[string]interface{}); ok {
				recommend, _ := m["Recommend"]
				log.Println("[SMS] url: ", recommend)
			}
			_, _err = util.AssertAsString(errorTem.Message)
			if _err != nil {
				log.Println("[SMS] err: ", _err, " phone numbers: ", smsReq.PhoneNumbers)
				continue
			}
		}
	}

	return
}

// 发送短信报警恢复
func sendSmsRecover(ctx *gin.Context, a *commonmodel.AlarmMQModel) (_err error) {

	// 获取报警消息对应的, 客户端
	smsTem := getSmsTem(ctx, a)
	if smsTem == nil || smsTem.CurClient == nil {
		log.Println("[ALARM] send alarm fail, client not exist")
		return
	}

	// 报警用户填充, 分批 -> 批量发送
	for _, smsReq := range getAlarmSmsRequest(ctx, a) {

		// 其他数据填充
		smsReq.SetSignName(smsTem.SignName)

		// 报警恢复模板
		smsReq.SetTemplateCode(smsTem.RecoverTemplateCode)
		smsReq.SetTemplateParam("{\"" + smsTem.RecoverSite + "\":\"" + a.SiteName + "\",\"" +
			smsTem.RecoverEquipment + "\":\"" + a.EquipmentName + "\",\"" +
			smsTem.RecoverCoding + "\":\"" + a.AlarmCode + "\",\"" +
			smsTem.RecoverValue + "\":\"" + fmt.Sprint(a.AlarmValue) + "\"}")

		// 发送并接受返回结果
		tryErr := func() (_e error) {
			defer func() {
				if r := tea.Recover(recover()); r != nil {
					_e = r
				}
			}()

			// 复制代码运行请自行打印 API 的返回值
			_, _err = smsTem.CurClient.SendSmsWithOptions(smsReq, &util.RuntimeOptions{})
			if _err != nil {
				return _err
			}
			return nil
		}()

		// TODO: 问题处理
		if tryErr != nil {
			var errorTem = &tea.SDKError{}
			if _t, ok := tryErr.(*tea.SDKError); ok {
				errorTem = _t
			} else {
				errorTem.Message = tea.String(tryErr.Error())
			}
			// 此处仅做打印展示，请谨慎对待异常处理，在工程项目中切勿直接忽略异常。
			// 错误 message
			log.Println("[SMS] recover error: ", tea.StringValue(errorTem.Message))
			// 诊断地址
			var data interface{}
			d := json.NewDecoder(strings.NewReader(tea.StringValue(errorTem.Data)))
			d.Decode(&data)
			if m, ok := data.(map[string]interface{}); ok {
				recommend, _ := m["Recommend"]
				log.Println("[SMS] recover url: ", recommend)
			}
			_, _err = util.AssertAsString(errorTem.Message)
			if _err != nil {
				log.Println("[SMS] recover err: ", _err, " phone numbers: ", smsReq.PhoneNumbers)
				continue
			}
		}
	}

	return
}

// 获取客户端 - 从初始化结果中获取, 对应报警设备, 部门, 租户的客户端(区分隔离鉴权)
func getSmsTem(ctx *gin.Context, a *commonmodel.AlarmMQModel) (smsTem *SmsTem) {
	scopeKey := getSmsScopeKey(ctx)
	corpSms := getSmsClients(scopeKey)
	if corpSms == nil && len(scopeKey) > 0 {
		if err := InitAllSmsClient(ctx); err != nil {
			log.Println("[ALARM] initialize renter sms client fail: ", err)
		}
		corpSms = getSmsClients(scopeKey)
	}
	if len(scopeKey) == 0 {
		log.Println("[ALARM] default renter data use")
	}

	// 获取部门客户端
	if len(corpSms) == 0 {
		// 不存在, 直接返回
		log.Println("[ALARM] getSmsTem nil data, need check")
		return nil
	} else {
		corpId := a.CorpId
		if corpId == 0 {
			// 默认部门客户端
			log.Println("[ALARM] default client use")
			return corpSms[fmt.Sprint(corpId)]
		}

		// 一个父部门一个父部门地去找
		for {
			smsTem = corpSms[fmt.Sprint(corpId)]
			if smsTem != nil {
				// 找到了, 对应的客户端
				log.Println("[ALARM] exact client matched")
				return
			}

			corpId = service.FindParentCorpOneByOne(ctx, corpId)
			if corpId == 0 {
				// 默认部门客户端
				log.Println("[ALARM] default client use")
				return corpSms[fmt.Sprint(corpId)]
			}
		}
	}
}

// 封装短信请求数据
func getAlarmSmsRequest(ctx *gin.Context, a *commonmodel.AlarmMQModel) (res []*dysmsapi20170525.SendSmsRequest) {

	userDatas := getAlarmUsers(ctx, a)
	if len(userDatas) == 0 {
		log.Println("[ALARM] find user fail, empty users")
		return
	}

	phoneNumbers := make([]string, 0, len(userDatas))
	for _, user := range userDatas {
		phoneNumbers = append(phoneNumbers, user.Phone)
	}
	return buildAlarmSmsRequests(phoneNumbers)
}

func buildAlarmSmsRequests(phoneNumbers []string) []*dysmsapi20170525.SendSmsRequest {
	const batchSize = 50
	requests := make([]*dysmsapi20170525.SendSmsRequest, 0, (len(phoneNumbers)+batchSize-1)/batchSize)
	for start := 0; start < len(phoneNumbers); start += batchSize {
		end := min(start+batchSize, len(phoneNumbers))
		requests = append(requests, &dysmsapi20170525.SendSmsRequest{
			PhoneNumbers: tea.String(strings.Join(phoneNumbers[start:end], ",")),
		})
	}
	return requests
}
