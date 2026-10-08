package dao

import (
	"dicloud/internal/tolink/model"
	"dicloud/util/gormutil"
	snowflakeutil "dicloud/util/snowflake"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
)

type ModuleTtdcDAO struct {
	db *gorm.DB
}

// 创建新的 GatewayDAO 实例
func NewModuleTtdcDAO(ctx *gin.Context) *ModuleTtdcDAO {
	return &ModuleTtdcDAO{db: gormutil.GetGormDB(ctx, "")}
}

func writeTtdcConfigRevisionIfAvailable(tx *gorm.DB, configType, targetKey, operation string) error {
	if !tx.Migrator().HasTable(&model.TtdcConfigRevision{}) {
		return nil
	}
	return tx.Create(&model.TtdcConfigRevision{ConfigType: configType, TargetKey: targetKey, Operation: operation}).Error
}

func ttdcTemplateSchemaReady(db *gorm.DB) bool {
	if db == nil || !db.Migrator().HasTable(&model.TtdcSchemaVersionRecord{}) {
		return false
	}
	var state model.TtdcSchemaVersionRecord
	if db.Where("component = ? AND version = ?", model.TtdcSchemaComponent, model.TtdcSchemaVersion).First(&state).Error != nil {
		return false
	}
	return model.TtdcSchemaRuntimeUsable(state.Status)
}

// 查询网关所有通道模块
func (dao *ModuleTtdcDAO) FindModule(g model.Gateway) (model.Gateway, bool) {

	// 查询所有
	var modules = []model.GatewayModuleTtdc{}
	dao.db.Model(&model.GatewayModuleTtdc{}).Preload("TtdcConfig").Preload("ModuleDatas").Where("gateway_id = ?", g.GatewayId).Find(&modules)
	if len(modules) > 0 {
		moduleIDs := make([]uint, 0, len(modules))
		for _, module := range modules {
			moduleIDs = append(moduleIDs, module.ID)
		}
		var bindings []model.GatewayTtdcTemplateBinding
		if dao.db.Migrator().HasTable(&model.GatewayTtdcTemplateBinding{}) {
			dao.db.Where("module_id IN ?", moduleIDs).Find(&bindings)
		}
		bindingByModule := make(map[uint]model.GatewayTtdcTemplateBinding, len(bindings))
		for _, binding := range bindings {
			bindingByModule[binding.ModuleID] = binding
		}
		dataTem := []any{}
		for _, v := range modules {
			if binding, ok := bindingByModule[v.ID]; ok {
				v.TemplateID = binding.TemplateID
				v.TemplateVersion = binding.TemplateVersion
				v.ProtocolCode = binding.ProtocolCode
				v.ParserProfile = binding.ParserProfile
			}
			dataTem = append(dataTem, v)
		}
		g.GatewayModules = dataTem
	}

	return g, true
}

// 是否存在标题
func (dao *ModuleTtdcDAO) ExistName(g model.GatewayModuleTtdc) (model.GatewayModuleTtdc, bool) {
	c := model.GatewayModuleTtdc{}

	// 查ttdc模块
	err := dao.db.Model(&model.GatewayModuleTtdc{}).Where("gateway_id = ? and name = ?", g.GatewayId, g.Name).Scan(&c).Error
	if err != nil || c.ID == 0 {
		return c, false
	}

	return c, true
}

// 创建通道模块
func (dao *ModuleTtdcDAO) CreateModule(gm model.GatewayModuleTtdc) (model.GatewayModuleTtdc, int64, int64, error) {

	// 开启事务
	tx := dao.db.Begin()
	if gm.TemplateID > 0 {
		adr, validADR := model.NormalizeTtdcADR(gm.TtdcConfig.ModuleAdr)
		if !validADR {
			tx.Rollback()
			return gm, 0, 0, errors.New("TTDC地址必须是00到FF的两位十六进制")
		}
		gm.TtdcConfig.ModuleAdr = adr
	}
	var selectedTemplate model.TtdcTemplate
	var selectedGroup model.TtdcTemplateGroup
	if gm.TemplateID > 0 {
		if !ttdcTemplateSchemaReady(tx) {
			tx.Rollback()
			return gm, 0, 0, errors.New("TTDC模板组数据库结构尚未完成初始化，不能创建公共模板通道")
		}
		if err := tx.Preload("Points").First(&selectedTemplate, gm.TemplateID).Error; err != nil {
			tx.Rollback()
			return gm, 0, 0, err
		}
		if gm.TemplateVersion <= 0 || gm.TemplateVersion != selectedTemplate.Version {
			tx.Rollback()
			return gm, 0, 0, errors.New("公共模板版本已变化，请重新选择模板")
		}
		if err := tx.First(&selectedGroup, selectedTemplate.GroupID).Error; err != nil {
			tx.Rollback()
			return gm, 0, 0, err
		}
		if !selectedGroup.Enabled || !selectedTemplate.Enabled {
			tx.Rollback()
			return gm, 0, 0, errors.New("模板组或模板已停止手工选择")
		}
		if err := applyTemplatePointMetadata(&gm, selectedTemplate); err != nil {
			tx.Rollback()
			return gm, 0, 0, err
		}
	}

	log.Println("===== 开始创建 GatewayModuleTtdc 数据 =====")
	log.Printf("原始数据: %+v\n", gm)

	// 1. 先创建主记录（如果需要）
	if err := tx.Model(&gm).Create(&gm).Error; err != nil || gm.ID == 0 {
		tx.Rollback()
		log.Printf("创建主记录失败: %v\n", err)
		return gm, 0, 0, err
	}

	// 2. 创建配置信息
	gm.TtdcConfig.ModuleId = int(gm.ID)
	if err := tx.Model(&model.ModuleTtdcConfig{}).Create(&gm.TtdcConfig).Error; err != nil || gm.TtdcConfig.ID == 0 {
		tx.Rollback()
		log.Printf("创建主记录配置参数失败: %v\n", err)
		return gm, 0, 0, err
	}

	// 3. 创建数据点
	var scsCount, dupCount int64
	if len(gm.ModuleDatas) > 0 {
		log.Printf("准备创建 %d 个 ModuleData + GatewayData 数据\n", len(gm.ModuleDatas))

		for i := range gm.ModuleDatas {
			group := &gm.ModuleDatas[i] // 使用指针修改原始数据

			// 自动生成dataId
			if group.DataId == "" && gm.TemplateID > 0 {
				tx.Rollback()
				return gm, scsCount, dupCount, errors.New("公共模板通道的数据点ID不能为空")
			}
			if group.DataId == "" {
				dataId := snowflakeutil.NextID()
				group.DataId = fmt.Sprint(dataId)
			}

			// 设置关联字段
			group.ModuleId = int(gm.ID)

			mdTem := model.ModuleTtdcData{}
			tx.Model(&mdTem).Where("module_id = ? AND name = ?", gm.ID, group.Name).Find(&mdTem)
			gdTem := model.GatewayData{}
			tx.Model(&gdTem).Where("gateway_id = ? AND data_id = ?", gm.GatewayId, group.DataId).Find(&gdTem)
			if mdTem.ID > 0 || gdTem.ID > 0 {
				dupCount++
				if gm.TemplateID > 0 {
					tx.Rollback()
					return gm, scsCount, dupCount, fmt.Errorf("公共模板数据点%s与现有数据冲突", group.DataId)
				}
				log.Println("Module already exist data", group.Name)
				continue
			}

			// 创建 Module通道 数据点
			if err := tx.Model(&group).Create(&group).Error; err != nil {
				log.Printf("创建 ModuleData[%d] 失败: %v\nSQL: %s\n",
					i, err,
					tx.Statement.SQL.String())
				tx.Rollback()
				return gm, 0, dupCount, err
			}

			// 创建 GatewayData 数据点
			if err := tx.Model(&model.GatewayData{}).Create(&model.GatewayData{GatewayId: gm.GatewayId, DataId: group.DataId, DataType: group.DataType, Name: group.Name}).Error; err != nil {
				log.Printf("创建 GatewayData[%d] 失败: %v\nSQL: %s\n",
					i, err,
					tx.Statement.SQL.String())
				tx.Rollback()
				return gm, 0, dupCount, err
			}

			// 成功个数
			scsCount++
		}
	}
	if gm.TemplateID > 0 && len(gm.ModuleDatas) == 0 {
		tx.Rollback()
		return gm, scsCount, dupCount, errors.New("公共模板通道的数据点列表不能为空")
	}
	if gm.TemplateID > 0 {
		if err := ensureManualAddressGroupBinding(tx, gm, selectedGroup); err != nil {
			tx.Rollback()
			return gm, scsCount, dupCount, err
		}
		binding := newManualTemplateBinding(gm, selectedGroup, selectedTemplate)
		if err := tx.Create(&binding).Error; err != nil {
			tx.Rollback()
			return gm, scsCount, dupCount, err
		}
	}
	if err := writeTtdcConfigRevisionIfAvailable(tx, "module", gm.GatewayId, "create"); err != nil {
		tx.Rollback()
		return gm, scsCount, dupCount, err
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("事务提交失败: %v\n", err)
		return gm, scsCount, dupCount, err
	}

	log.Println("===== 数据创建成功 =====")
	return gm, scsCount, dupCount, nil

}

func applyTemplatePointMetadata(gm *model.GatewayModuleTtdc, template model.TtdcTemplate) error {
	adr := strings.ToUpper(strings.TrimSpace(gm.TtdcConfig.ModuleAdr))
	points := make(map[string]model.TtdcTemplatePoint, len(template.Points))
	for _, point := range template.Points {
		if !point.Enabled {
			continue
		}
		points[adr+"_"+point.BaseDataID] = point
	}
	seen := make(map[string]bool, len(gm.ModuleDatas))
	for i := range gm.ModuleDatas {
		dataID := strings.TrimSpace(gm.ModuleDatas[i].DataId)
		point, ok := points[dataID]
		if !ok {
			return fmt.Errorf("数据点%s不属于所选公共模板", gm.ModuleDatas[i].DataId)
		}
		if seen[dataID] {
			return fmt.Errorf("数据点%s重复", dataID)
		}
		seen[dataID] = true
		gm.ModuleDatas[i].AlgorithmKey = point.AlgorithmKey
	}
	if len(seen) != len(points) {
		return fmt.Errorf("公共模板数据点不完整：应提交%d个启用点位，实际%d个", len(points), len(seen))
	}
	return nil
}

func ensureManualAddressGroupBinding(tx *gorm.DB, gm model.GatewayModuleTtdc, group model.TtdcTemplateGroup) error {
	adr, validADR := model.NormalizeTtdcADR(gm.TtdcConfig.ModuleAdr)
	if !validADR {
		return errors.New("TTDC地址必须是00到FF的两位十六进制")
	}
	var existing model.GatewayTtdcAddressGroupBinding
	err := tx.Where("gateway_id = ? AND module_adr = ?", gm.GatewayId, adr).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row := model.GatewayTtdcAddressGroupBinding{GatewayID: gm.GatewayId, ModuleADR: adr, TemplateGroupID: group.ID, GroupCode: group.GroupCode, TopicSegment: group.TopicSegment, ModuleSOI: gm.TtdcConfig.ModuleSoi, ModuleVER: gm.TtdcConfig.ModuleVer, ModuleCID1: gm.TtdcConfig.ModuleCid1, BindSource: "manual", BindingStatus: "active"}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return writeTtdcConfigRevisionIfAvailable(tx, "address_group", gm.GatewayId+":"+adr, "create")
	}
	if err != nil {
		return err
	}
	if existing.BindingStatus == "inactive" {
		var existingBindings int64
		if err := tx.Model(&model.GatewayTtdcTemplateBinding{}).Where("gateway_id = ? AND module_adr = ?", gm.GatewayId, adr).Count(&existingBindings).Error; err != nil {
			return err
		}
		if existingBindings > 0 {
			return errors.New("历史停用ADR仍有残留通道，不能直接恢复")
		}
		if err := tx.Model(&existing).Updates(map[string]interface{}{
			"template_group_id": group.ID,
			"group_code":        group.GroupCode,
			"topic_segment":     group.TopicSegment,
			"module_soi":        gm.TtdcConfig.ModuleSoi,
			"module_ver":        gm.TtdcConfig.ModuleVer,
			"module_cid1":       gm.TtdcConfig.ModuleCid1,
			"bind_source":       "manual",
			"binding_status":    "active",
		}).Error; err != nil {
			return err
		}
		return writeTtdcConfigRevisionIfAvailable(tx, "address_group", gm.GatewayId+":"+adr, "reactivate")
	}
	if existing.TemplateGroupID != group.ID || existing.TopicSegment != group.TopicSegment {
		return errors.New("当前网关ADR已经绑定其他电池模板组")
	}
	if existing.ModuleSOI != gm.TtdcConfig.ModuleSoi || existing.ModuleVER != gm.TtdcConfig.ModuleVer || existing.ModuleCID1 != gm.TtdcConfig.ModuleCid1 {
		return errors.New("当前网关ADR的SOI、VER或CID1与历史绑定冲突")
	}
	return nil
}

func newManualTemplateBinding(gm model.GatewayModuleTtdc, group model.TtdcTemplateGroup, template model.TtdcTemplate) model.GatewayTtdcTemplateBinding {
	return model.GatewayTtdcTemplateBinding{ModuleID: gm.ID, GatewayID: gm.GatewayId, TemplateGroupID: group.ID, TemplateID: template.ID, TemplateCode: template.TemplateCode, TemplateVersion: template.Version, ProtocolCode: template.ProtocolCode, ParserProfile: template.ParserProfile, ModuleSOI: gm.TtdcConfig.ModuleSoi, ModuleVER: gm.TtdcConfig.ModuleVer, ModuleADR: strings.ToUpper(gm.TtdcConfig.ModuleAdr), ModuleCID1: gm.TtdcConfig.ModuleCid1, NormalizedCID2: template.CID2Type, CreateSource: "manual"}
}

// 删除单个模块
func (dao *ModuleTtdcDAO) DeleteModule(gm model.GatewayModuleTtdc) (model.GatewayModuleTtdc, bool) {

	// 开启事务
	tx := dao.db.Begin()

	// 1. 删除gateway data
	moduleDatas := []model.ModuleTtdcData{}
	err := tx.Model(&model.ModuleTtdcData{}).Where("module_id = ?", gm.ID).Find(&moduleDatas).Error
	if err != nil {
		tx.Rollback()
		log.Println("[ModuleTtdcDAO] search moduledata fail")
		return gm, false
	}
	if len(moduleDatas) > 0 {
		for _, dataTem := range moduleDatas {
			err := tx.Model(&model.GatewayData{}).Where("gateway_id = ? AND data_id = ?", gm.GatewayId, dataTem.DataId).Delete(&model.GatewayData{}).Error
			if err != nil {
				tx.Rollback()
				log.Println("[ModuleTtdcDAO] delete gatewaydatas fail", err)
				return gm, false
			}
		}
	}

	// 2. 删除关联配置
	err = tx.Model(&model.ModuleTtdcConfig{}).Where("module_id = ?", gm.ID).Delete(&model.ModuleTtdcConfig{}).Error
	if err != nil {
		tx.Rollback()
		log.Println("[ModuleTtdcDAO] delete module config fail: ", err)
		return gm, false
	}

	// 3. 删除通道数据
	err = tx.Model(&model.ModuleTtdcData{}).Where("module_id = ?", gm.ID).Delete(&model.ModuleTtdcData{}).Error
	if err != nil {
		tx.Rollback()
		log.Println("[ModuleTtdcDAO] delete module data fail: ", err)
		return gm, false
	}

	// 4. 删除公共模板绑定
	if tx.Migrator().HasTable(&model.GatewayTtdcTemplateBinding{}) {
		err = tx.Where("module_id = ?", gm.ID).Delete(&model.GatewayTtdcTemplateBinding{}).Error
		if err != nil {
			tx.Rollback()
			log.Println("[ModuleTtdcDAO] delete template binding fail: ", err)
			return gm, false
		}
	}
	// 5. 删除通道
	err = tx.Model(&gm).Delete(gm).Error
	if err != nil {
		tx.Rollback()
		log.Println("[ModuleTtdcDAO] delete module fail: ", err)
		return gm, false
	}

	if err = writeTtdcConfigRevisionIfAvailable(tx, "module", gm.GatewayId, "delete"); err != nil {
		tx.Rollback()
		return gm, false
	}

	// 6. 提交事务
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		log.Println("[ModuleTtdcDAO] commit delete fail: ", err)
		return gm, false
	}

	return gm, true
}

// 根据网关id删除所有数据
func (dao *ModuleTtdcDAO) DeleteModuleByGatewayId(gatewayId string) error {

	// 开启事务
	tx := dao.db.Begin()

	log.Printf("\n===== 开始删除网关 %s 的所有Drive数据 =====\n", gatewayId)

	// 1. 查询该网关下的所有Drive组
	var groups []model.GatewayModuleTtdc
	if err := tx.Model(&model.GatewayModuleTtdc{}).Where("gateway_id = ?", gatewayId).Find(&groups).Error; err != nil {
		log.Printf("查询Drive组失败: %v\n", err)
		tx.Rollback()
		return fmt.Errorf("查询Drive组失败: %w", err)
	}

	for _, group := range groups {

		// 2. 遍历并删除关联的Data数据
		// 删除关联的ModuleTtdcData
		if err := tx.Model(&model.ModuleTtdcData{}).Where("module_id = ?", group.ID).Delete(&model.ModuleTtdcData{}).Error; err != nil {
			log.Printf("[ModuleTtdcDAO] 删除Group[%d]的Data数据失败: %v\n", group.ID, err)
			tx.Rollback()
			return fmt.Errorf("删除Data数据失败: %w", err)
		}

		// 3. 删除所有Config数据
		if err := tx.Model(&model.ModuleTtdcConfig{}).Where("module_id = ?", group.ID).Delete(&model.ModuleTtdcConfig{}).Error; err != nil {
			log.Printf("[ModuleTtdcDAO] 删除Group[%d]的config数据失败: %v\n", group.ID, err)
			tx.Rollback()
			return fmt.Errorf("删除config数据失败: %w", err)
		}
	}
	if tx.Migrator().HasTable(&model.GatewayTtdcTemplateBinding{}) {
		if err := tx.Where("gateway_id = ?", gatewayId).Delete(&model.GatewayTtdcTemplateBinding{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("删除模板绑定失败: %w", err)
		}
	}
	if tx.Migrator().HasTable(&model.GatewayTtdcAddressGroupBinding{}) {
		var addresses []model.GatewayTtdcAddressGroupBinding
		if err := tx.Where("gateway_id = ?", gatewayId).Find(&addresses).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("查询ADR模板组绑定失败: %w", err)
		}
		// Deleting a TTDC gateway releases every ADR completely. Keeping inactive
		// ownership here would prevent the same serial number from later binding
		// that ADR to another battery template group.
		if err := tx.Where("gateway_id = ?", gatewayId).Delete(&model.GatewayTtdcAddressGroupBinding{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("删除ADR模板组绑定失败: %w", err)
		}
		for _, address := range addresses {
			if err := writeTtdcConfigRevisionIfAvailable(tx, "address_group", gatewayId+":"+address.ModuleADR, "delete"); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// 4. 删除所有 DriveGroup
	if err := tx.Model(&model.GatewayModuleTtdc{}).Where("gateway_id = ?", gatewayId).Delete(&model.GatewayModuleTtdc{}).Error; err != nil {
		log.Printf("[ModuleTtdcDAO] 删除gatewayId[%s]的Drive组失败: %v\n", gatewayId, err)
		tx.Rollback()
		return fmt.Errorf("删除Drive组失败: %w", err)
	}

	// 5. 删除所有 GatewayData
	if err := tx.Model(&model.GatewayData{}).Where("gateway_id = ?", gatewayId).Delete(&model.GatewayData{}).Error; err != nil {
		log.Printf("[ModuleTtdcDAO] 删除gatewayId[%s]的GatewayDatas失败: %v\n", gatewayId, err)
		tx.Rollback()
		return fmt.Errorf("删除GatewayDatas失败: %w", err)
	}

	if err := writeTtdcConfigRevisionIfAvailable(tx, "module", gatewayId, "delete"); err != nil {
		tx.Rollback()
		return fmt.Errorf("写入配置修订失败: %w", err)
	}

	// 6. 提交数据
	if err := tx.Commit().Error; err != nil {
		log.Println("[ModuleTtdcDAO] 事务提交失败: ", err)
		return fmt.Errorf("事务提交失败: %w", err)
	}

	log.Println("===== 数据删除成功 =====\n..\n..")
	return nil
}

// 根据id查询数据
func (dao *ModuleTtdcDAO) ExistModuleId(gmd model.GatewayModuleTtdc) (model.GatewayModuleTtdc, bool) {
	dataTem := model.GatewayModuleTtdc{}

	err := dao.db.Model(&dataTem).Where("id = ?", gmd.ID).First(&dataTem).Error
	if err != nil || dataTem.ID == 0 {
		log.Println("[ModuleTtdcDAO] find drive by id fail", gmd)
		return dataTem, false
	}

	return dataTem, true
}

// 操作通道数据 -
func (dao *ModuleTtdcDAO) UpdateModule(funcStr string, gm model.GatewayModuleTtdc) (model.GatewayModuleTtdc, bool) {

	switch {

	case model.MODULE_UPDATE == funcStr:
		{
			// 修改module, 修改增加module data
			return updateModuleTtdc(dao, gm)
		}

	case funcStr == "deletepoint":
		{
			return deleteModuleTtdcPoint(dao, gm)
		}

	default:
		// 默认更新数据
		return updateModuleTtdc(dao, gm)
	}
}

// 修改通道, 新增/修改通道内数据点
func updateModuleTtdc(dao *ModuleTtdcDAO, gm model.GatewayModuleTtdc) (model.GatewayModuleTtdc, bool) {

	// 开启事务
	tx := dao.db.Begin()
	var lockedBinding model.GatewayTtdcTemplateBinding
	bindingTableExists := tx.Migrator().HasTable(&model.GatewayTtdcTemplateBinding{})
	bound := false
	if bindingTableExists {
		bound = tx.Where("module_id = ?", gm.ID).First(&lockedBinding).Error == nil
	}
	if (bound || gm.TemplateID > 0) && !ttdcTemplateSchemaReady(tx) {
		tx.Rollback()
		log.Println("[ModuleTtdcDAO] TTDC template-group schema is not initialized")
		return gm, false
	}
	if bound {
		adr, validADR := model.NormalizeTtdcADR(gm.TtdcConfig.ModuleAdr)
		if !validADR {
			tx.Rollback()
			log.Println("[ModuleTtdcDAO] invalid TTDC ADR")
			return gm, false
		}
		gm.TtdcConfig.ModuleAdr = adr
		if gm.TemplateID != lockedBinding.TemplateID || gm.TemplateVersion != lockedBinding.TemplateVersion || gm.TtdcConfig.ModuleSoi != lockedBinding.ModuleSOI || gm.TtdcConfig.ModuleVer != lockedBinding.ModuleVER || !strings.EqualFold(gm.TtdcConfig.ModuleAdr, lockedBinding.ModuleADR) || gm.TtdcConfig.ModuleCid1 != lockedBinding.ModuleCID1 {
			tx.Rollback()
			log.Println("[ModuleTtdcDAO] template-bound channel identity or template snapshot cannot be modified")
			return gm, false
		}
	}
	if gm.TemplateID > 0 && !bound {
		adr, validADR := model.NormalizeTtdcADR(gm.TtdcConfig.ModuleAdr)
		if !validADR {
			tx.Rollback()
			log.Println("[ModuleTtdcDAO] invalid TTDC ADR")
			return gm, false
		}
		gm.TtdcConfig.ModuleAdr = adr
	}
	var selectedTemplate model.TtdcTemplate
	var selectedGroup model.TtdcTemplateGroup
	if gm.TemplateID > 0 && !bound {
		if err := tx.Preload("Points").First(&selectedTemplate, gm.TemplateID).Error; err != nil {
			tx.Rollback()
			return gm, false
		}
		if err := tx.First(&selectedGroup, selectedTemplate.GroupID).Error; err != nil {
			tx.Rollback()
			return gm, false
		}
		if !selectedGroup.Enabled || !selectedTemplate.Enabled {
			tx.Rollback()
			return gm, false
		}
		if gm.TemplateVersion <= 0 || gm.TemplateVersion != selectedTemplate.Version {
			tx.Rollback()
			return gm, false
		}
		if err := applyTemplatePointMetadata(&gm, selectedTemplate); err != nil {
			tx.Rollback()
			return gm, false
		}
	}

	// 更新通道数据
	var err error
	if bound {
		err = tx.Model(&model.GatewayModuleTtdc{}).Where("id = ?", gm.ID).Updates(map[string]interface{}{"name": gm.Name, "title": gm.Title, "transport": gm.Transport}).Error
	} else {
		err = tx.Updates(&gm).Error
	}
	if err != nil {
		tx.Rollback()
		log.Println("[ModuleTtdcDAO] UpdateModule fail", err)
		return gm, false
	}

	// 公共模板通道按页面提交的完整列表同步，切换模板时不会残留旧点位。
	if gm.TemplateID > 0 {
		if len(gm.ModuleDatas) == 0 {
			tx.Rollback()
			log.Println("[ModuleTtdcDAO] template-bound channel has no data points")
			return gm, false
		}
		var existingPoints []model.ModuleTtdcData
		if err = tx.Where("module_id = ?", gm.ID).Find(&existingPoints).Error; err != nil {
			tx.Rollback()
			return gm, false
		}
		existingByID := make(map[uint]model.ModuleTtdcData, len(existingPoints))
		incomingIDs := make(map[uint]bool, len(gm.ModuleDatas))
		for _, point := range existingPoints {
			existingByID[point.ID] = point
		}
		for _, point := range gm.ModuleDatas {
			if strings.TrimSpace(point.DataId) == "" {
				tx.Rollback()
				log.Println("[ModuleTtdcDAO] template-bound point dataId is empty")
				return gm, false
			}
			if point.ID > 0 {
				if _, ok := existingByID[point.ID]; !ok {
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] submitted point does not belong to module", point.ID)
					return gm, false
				}
				incomingIDs[point.ID] = true
			}
		}
		for _, point := range existingPoints {
			if incomingIDs[point.ID] {
				continue
			}
			if err = tx.Where("gateway_id = ? AND data_id = ?", gm.GatewayId, point.DataId).Delete(&model.GatewayData{}).Error; err != nil {
				tx.Rollback()
				return gm, false
			}
			if err = tx.Delete(&point).Error; err != nil {
				tx.Rollback()
				return gm, false
			}
		}
	}

	// 更新通道内数据点
	if len(gm.ModuleDatas) > 0 {
		for _, v := range gm.ModuleDatas {
			if v.ID > 0 {
				// 更新数据点
				var vTem model.ModuleTtdcData
				if err = tx.Where("id = ? AND module_id = ?", v.ID, gm.ID).First(&vTem).Error; err != nil {
					// 查是否存在
					log.Println("[ModuleTtdcDAO] UpdateModule moduledata not exist", err)
					tx.Rollback()
					return gm, false
				}
				if strings.TrimSpace(v.DataId) == "" {
					v.DataId = vTem.DataId
				}
				if gm.TemplateID > 0 && v.DataId != vTem.DataId {
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] template-bound point dataId cannot be modified", vTem.DataId)
					return gm, false
				}
				if bound && !sameOptionalString(v.AlgorithmKey, vTem.AlgorithmKey) {
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] template-bound point algorithmKey cannot be modified", vTem.DataId)
					return gm, false
				}
				// 更新 GatewayData
				err = tx.Model(&model.GatewayData{}).Where("gateway_id = ? AND data_id = ?", gm.GatewayId, vTem.DataId).Updates(map[string]interface{}{"data_id": v.DataId, "name": v.Name, "data_type": v.DataType}).Error
				if err != nil {
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] UpdateModule update GatewayData fail", err)
					return gm, false
				}
				// 更新 ModuleData 数据点
				v.ModuleId = int(gm.ID)
				err = tx.Model(&model.ModuleTtdcData{}).Where("id = ? AND module_id = ?", v.ID, gm.ID).Updates(map[string]interface{}{"data_id": v.DataId, "name": v.Name, "data_type": v.DataType, "transport": v.Transport, "formula": v.Formula, "unformula": v.Unformula, "algorithm_key": v.AlgorithmKey}).Error
				if err != nil {
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] UpdateModule update moduledata fail", err)
					return gm, false
				}
			} else {
				// 插入数据点
				var vTem model.ModuleTtdcData
				if bound && v.AlgorithmKey != nil {
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] ordinary editing cannot assign algorithmKey to a new bound point")
					return gm, false
				}
				if v.DataId == "" && gm.TemplateID > 0 {
					tx.Rollback()
					return gm, false
				}
				if v.DataId == "" {
					dataId := snowflakeutil.NextID()
					v.DataId = fmt.Sprint(dataId)
				}
				tx.Model(&v).Limit(1).Where("module_id = ? and (data_id = ? or name = ?)", gm.ID, v.DataId, v.Name).Scan(&vTem)

				var vTem3 model.GatewayData
				tx.Model(&model.GatewayData{}).Limit(1).Where("gateway_id = ? and data_id = ?", gm.GatewayId, v.DataId).Scan(&vTem3)
				if vTem.ID > 0 || vTem3.ID > 0 {
					// 查重
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] UpdateModule insert module data already exist dataId or name or GatewayDataId fail: \n1: ", vTem, " \n3: ", vTem3)
					return gm, false
				}

				// 自动生成dataId
				if v.DataId == "" {
					dataId := snowflakeutil.NextID()
					v.DataId = fmt.Sprint(dataId)
				}
				// 创建
				err := tx.Model(&model.GatewayData{}).Create(&model.GatewayData{GatewayId: gm.GatewayId, DataId: v.DataId, DataType: v.DataType, Name: v.Name}).Error
				if err != nil {
					// 插入 GatewayData
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] UpdateModule insert GatewayData fail", err)
					return gm, false
				}
				v.ModuleId = int(gm.ID)
				err = tx.Model(&v).Create(&v).Error
				if err != nil {
					// 插入 ModuleData
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] UpdateModule insert moduledata fail", err)
					return gm, false
				}
			}
		}
	}

	// 更新通道配置信息
	if gm.TtdcConfig.ID > 0 && !bound {
		err := tx.Updates(&gm.TtdcConfig).Error
		if err != nil {
			tx.Rollback()
			log.Println("[ModuleTtdcDAO] update module ttdc config fail ", err)
			return gm, false
		}
	}
	if gm.TemplateID > 0 {
		var existing model.GatewayTtdcTemplateBinding
		bindingErr := tx.Where("module_id = ?", gm.ID).First(&existing).Error
		preserveSnapshot := bindingErr == nil && existing.TemplateID == gm.TemplateID && existing.TemplateVersion == gm.TemplateVersion
		var binding model.GatewayTtdcTemplateBinding
		if preserveSnapshot {
			binding = existing
			binding.GatewayID = gm.GatewayId
			binding.ModuleSOI = gm.TtdcConfig.ModuleSoi
			binding.ModuleVER = gm.TtdcConfig.ModuleVer
			binding.ModuleADR = gm.TtdcConfig.ModuleAdr
			binding.ModuleCID1 = gm.TtdcConfig.ModuleCid1
		} else {
			if gm.TemplateVersion <= 0 || gm.TemplateVersion != selectedTemplate.Version {
				tx.Rollback()
				log.Println("[ModuleTtdcDAO] submitted template version is stale")
				return gm, false
			}
			if err := ensureManualAddressGroupBinding(tx, gm, selectedGroup); err != nil {
				tx.Rollback()
				return gm, false
			}
			binding = newManualTemplateBinding(gm, selectedGroup, selectedTemplate)
		}
		if errors.Is(bindingErr, gorm.ErrRecordNotFound) {
			if err = tx.Create(&binding).Error; err != nil {
				tx.Rollback()
				return gm, false
			}
		} else if bindingErr == nil {
			if existing.CreateSource != "" {
				binding.CreateSource = existing.CreateSource
			}
			if err = tx.Model(&existing).Updates(&binding).Error; err != nil {
				tx.Rollback()
				return gm, false
			}
		} else {
			tx.Rollback()
			return gm, false
		}
	}
	if err := writeTtdcConfigRevisionIfAvailable(tx, "module", gm.GatewayId, "update"); err != nil {
		tx.Rollback()
		return gm, false
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Println("[ModuleTtdcDAO] commit update fail", err)
		return gm, false
	}

	return gm, true
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return strings.TrimSpace(*left) == strings.TrimSpace(*right)
}

// 删除通道内数据点
func deleteModuleTtdcPoint(dao *ModuleTtdcDAO, gm model.GatewayModuleTtdc) (model.GatewayModuleTtdc, bool) {

	// 开启事务
	tx := dao.db.Begin()
	// 删除通道内数据点
	if len(gm.ModuleDatas) > 0 {
		for _, v := range gm.ModuleDatas {
			if v.ID > 0 {
				vTem := model.ModuleTtdcData{ID: v.ID}
				tx.Model(&v).First(&vTem)
				if len(vTem.DataId) == 0 {
					// 查重
					log.Println("[ModuleTtdcDAO] deletePoint moduledata not exist", v)
					return gm, false
				}
				// 删除 GatewayData
				err := tx.Model(&model.GatewayData{}).Where("gateway_id = ? AND data_id = ?", gm.GatewayId, v.DataId).Delete(&model.GatewayData{GatewayId: gm.GatewayId, DataId: v.DataId}).Error
				if err != nil {
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] deletePoint delete GatewayData fail", err)
					return gm, false
				}
				// 更新 ModuleData 数据点
				err = tx.Model(&v).Delete(&v).Error
				if err != nil {
					tx.Rollback()
					log.Println("[ModuleTtdcDAO] deletePoint update moduledata fail", err)
					return gm, false
				}
			} else {
				// 请指定数据点
				log.Println("[ModuleTtdcDAO] deletePoint ID is null")
				return gm, false
			}

		}
	}

	if err := writeTtdcConfigRevisionIfAvailable(tx, "module", gm.GatewayId, "update"); err != nil {
		tx.Rollback()
		return gm, false
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return gm, false
	}

	return gm, true
}

type moduleTtdcPointTransportDTO struct {
	Name      string `bson:"name" json:"name"`
	DataId    string `bson:"dataId" json:"dataId"`
	DataType  int    `bson:"dataType" json:"dataType"`   // 1: 浮点型, 2: 字符型
	Formula   string `bson:"formula" json:"formula"`     // 处理公式
	Unformula string `bson:"unformula" json:"unformula"` // 逆处理公式
	TelType   uint32 `bson:"telType" json:"telType"`     // 遥测量/遥信量 1: 遥测量 2: 遥信量
	ByteOrder string `bson:"byteOrder" json:"byteOrder"` // 字节序 如: 1234 4321
	ByteNum   uint32 `bson:"byteNum" json:"byteNum"`     // 字节序号 1-...
	ByteCount uint32 `bson:"byteCount" json:"byteCount"` // 字节数 1, 2, ,3, 4
	DoBit     uint32 `bson:"doBit" json:"doBit"`         // 是否取位 0:否 1:是
	BitNum    uint32 `bson:"bitNum" json:"bitNum"`       // 取位位号 0-7 8-15...
}

var (
	moduleTtdcHeaderOptions = map[string]int{
		"名称":      -1,
		"数据类型":    -1,
		"处理公式":    -1,
		"逆处理公式":   -1,
		"遥测量/遥信量": -1,
		"字节序":     -1,
		"字节序号":    -1,
		"字节数":     -1,
		"是否取位":    -1,
		"取位位号":    -1,
	}
	moduleTtdcDataTypeOptions = map[int]string{
		1: "浮点型",
		2: "字符型",
	}
	moduleTtdcTelTypeOptions = map[int]string{
		1: "遥测量",
		2: "遥信量",
	}
	moduleTtdcByteOrderOptions = map[string]string{
		"1234": "四字节整数正序",
		"4321": "四字节整数逆序",
	}
	moduleTtdcByteNumOptions = map[int]string{
		1: "序号1",
		2: "序号2",
		3: "序号3",
		4: "序号4",
		5: "序号5",
		6: "序号6",
		7: "序号7",
		8: "序号8",
		9: "序号9",
	}
	moduleTtdcByteCountOptions = map[int]string{
		1: "1个字节",
		2: "2个字节",
	}
	moduleTtdcDoBitOptions = map[int]string{
		0: "不取位",
		1: "取位",
	}
	moduleTtdcBitNumOptions = map[int]string{
		0:  "取位[0]",
		1:  "取位[1]",
		2:  "取位[2]",
		3:  "取位[3]",
		4:  "取位[4]",
		5:  "取位[5]",
		6:  "取位[6]",
		7:  "取位[7]",
		8:  "取位[8]",
		9:  "取位[9]",
		10: "取位[10]",
		11: "取位[11]",
		12: "取位[12]",
		13: "取位[13]",
		14: "取位[14]",
		15: "取位[15]",
	}
)

// 导出Excel - 自定义参数数据 - 数据表
func (dao *ModuleTtdcDAO) ExportExcel(gmd model.GatewayModuleTtdc) (fileName string, err error) {

	pointList := []moduleTtdcPointTransportDTO{}

	myExcel := excelize.NewFile()

	myExcel.NewSheet("配置")
	myExcel.NewSheet("说明")
	myExcel.DeleteSheet("Sheet1")

	moduleTem, res := dao.ExistModuleId(gmd)
	if !res {
		return "", errors.New("通道不存在")
	}

	moduleDatas := []model.ModuleTtdcData{}
	dao.db.Model(&model.ModuleTtdcData{}).Where("module_id = ?", gmd.ID).Find(&moduleDatas)
	if len(moduleDatas) > 0 {

		switch moduleTem.Type {
		case 1:
			{
				for _idx := range moduleDatas {
					point := moduleTtdcPointTransportDTO{}
					point.Name = moduleDatas[_idx].Name
					point.DataId = moduleDatas[_idx].DataId
					point.DataType = moduleDatas[_idx].DataType
					point.Formula = moduleDatas[_idx].Formula
					point.Unformula = moduleDatas[_idx].Unformula
					fmt.Sscanf(
						moduleDatas[_idx].Transport,
						"%d %s %d %d %d",
						&point.TelType, &point.ByteOrder, &point.ByteNum,
						&point.ByteCount, &point.DoBit, &point.BitNum,
					)
					pointList = append(pointList, point)
				}

				myExcel.SetCellValue("配置", "A1", "名称")
				myExcel.SetCellValue("配置", "B1", "数据类型")
				myExcel.SetCellValue("配置", "C1", "处理公式")
				myExcel.SetCellValue("配置", "D1", "逆处理公式")
				myExcel.SetCellValue("配置", "E1", "遥测量/遥信量")
				myExcel.SetCellValue("配置", "F1", "字节序")
				myExcel.SetCellValue("配置", "G1", "字节序号")
				myExcel.SetCellValue("配置", "H1", "字节数")
				myExcel.SetCellValue("配置", "I1", "是否取位")
				myExcel.SetCellValue("配置", "J1", "取位位号")

				for idx := range pointList {
					myExcel.SetCellValue("配置", fmt.Sprintf("A%d", idx+2), pointList[idx].Name)
					myExcel.SetCellValue("配置", fmt.Sprintf("B%d", idx+2), moduleTtdcDataTypeOptions[pointList[idx].DataType])
					myExcel.SetCellValue("配置", fmt.Sprintf("C%d", idx+2), pointList[idx].Formula)
					myExcel.SetCellValue("配置", fmt.Sprintf("D%d", idx+2), pointList[idx].Unformula)
					myExcel.SetCellValue("配置", fmt.Sprintf("E%d", idx+2), moduleTtdcTelTypeOptions[int(pointList[idx].TelType)])
					myExcel.SetCellValue("配置", fmt.Sprintf("F%d", idx+2), moduleTtdcByteOrderOptions[pointList[idx].ByteOrder])
					myExcel.SetCellValue("配置", fmt.Sprintf("G%d", idx+2), moduleTtdcByteNumOptions[int(pointList[idx].ByteNum)])
					myExcel.SetCellValue("配置", fmt.Sprintf("H%d", idx+2), moduleTtdcByteCountOptions[int(pointList[idx].ByteCount)])
					myExcel.SetCellValue("配置", fmt.Sprintf("I%d", idx+2), moduleTtdcDoBitOptions[int(pointList[idx].DoBit)])
					myExcel.SetCellValue("配置", fmt.Sprintf("J%d", idx+2), moduleTtdcBitNumOptions[int(pointList[idx].BitNum)])
				}

				myExcel.SetRowHeight("配置", 1, 20)
				myExcel.SetColWidth("配置", "A", "A", 32)
				myExcel.SetColWidth("配置", "B", "F", 20)

				// style, _ := myExcel.NewStyle(`{"Font":{"Bold":true,"Size":12}}`)
				style, _ := myExcel.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}})
				myExcel.SetCellStyle("配置", "A1", "F1", style)

				// style2, _ := myExcel.NewStyle(`{"Alignment":{"Horizontal":"left"}}`)
				style2, _ := myExcel.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Horizontal: "left"}})
				myExcel.SetCellStyle("配置", "E2", fmt.Sprintf("H%d", len(pointList)+1), style2)
				myExcel.SetCellStyle("配置", "F2", fmt.Sprintf("I%d", len(pointList)+1), style2)

				myExcel.SetCellValue("说明", "A1", "名称")
				myExcel.SetCellValue("说明", "B1", "数据类型")
				myExcel.SetCellValue("说明", "C1", "处理公式")
				myExcel.SetCellValue("说明", "D1", "逆处理公式")
				myExcel.SetCellValue("说明", "E1", "遥测量/遥信量")
				myExcel.SetCellValue("说明", "F1", "字节序")
				myExcel.SetCellValue("说明", "G1", "字节序号")
				myExcel.SetCellValue("说明", "H1", "字节数")
				myExcel.SetCellValue("说明", "I1", "是否取位")
				myExcel.SetCellValue("说明", "J1", "取位位号")

				myExcel.SetRowHeight("说明", 1, 20)
				myExcel.SetColWidth("说明", "A", "A", 32)
				myExcel.SetColWidth("说明", "B", "F", 20)

				myExcel.SetCellStyle("说明", "A1", "F1", style)

				myExcel.SetCellValue("说明", "A2", "变量的名称，不可重复")
				for idx, val := range moduleTtdcDataTypeOptions {
					myExcel.SetCellValue("说明", fmt.Sprintf("B%d", idx+1), val)
				}
				myExcel.SetCellValue("说明", "C2", "例：(x + 15) / 6")
				myExcel.SetCellValue("说明", "D2", "例：x * 6 - 15")
				for idx, val := range moduleTtdcTelTypeOptions {
					// idx从1开始
					myExcel.SetCellValue("说明", fmt.Sprintf("E%d", idx+1), val)
				}
				idx := 0
				for _, val := range moduleTtdcByteOrderOptions {
					myExcel.SetCellValue("说明", fmt.Sprintf("F%d", idx+2), val)
				}
				for idx, val := range moduleTtdcByteNumOptions {
					// idx从1开始
					myExcel.SetCellValue("说明", fmt.Sprintf("G%d", idx+1), val)
				}
				for idx, val := range moduleTtdcByteCountOptions {
					// idx从1开始
					myExcel.SetCellValue("说明", fmt.Sprintf("H%d", idx+1), val)
				}
				for idx, val := range moduleTtdcDoBitOptions {
					myExcel.SetCellValue("说明", fmt.Sprintf("I%d", idx+2), val)
				}
				for idx, val := range moduleTtdcBitNumOptions {
					myExcel.SetCellValue("说明", fmt.Sprintf("J%d", idx+2), val)
				}
			}
		// case 2, 3:
		// 	{
		// 		for _idx := range moduleDatas {
		// 			point := pointTransport{}
		// 			point.Name = moduleDatas[_idx].Name
		// 			fmt.Sscanf(
		// 				moduleDatas[_idx].Transport,
		// 				"%f %d %d %d %s %s %d %d",
		// 				&point.UploadParm, &point.Function, &point.TypeData, &point.Address,
		// 				&point.Formula, &point.Unformula,
		// 				&point.UploadWay, &point.UploadParm2,
		// 			)
		// 			pointList = append(pointList, point)
		// 		}

		// 		myExcel.SetCellValue("配置", "A1", "名称")
		// 		myExcel.SetCellValue("配置", "B1", "功能号")
		// 		myExcel.SetCellValue("配置", "C1", "通道地址")
		// 		myExcel.SetCellValue("配置", "D1", "数据类型")
		// 		myExcel.SetCellValue("配置", "E1", "处理公式")
		// 		myExcel.SetCellValue("配置", "F1", "逆处理公式")
		// 		myExcel.SetCellValue("配置", "G1", "上报模式")
		// 		myExcel.SetCellValue("配置", "H1", "上报参数一")
		// 		myExcel.SetCellValue("配置", "I1", "上报参数二")

		// 		for idx := range pointList {
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("A%d", idx+2), pointList[idx].Name)
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("B%d", idx+2), functionOption[pointList[idx].Function])
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("C%d", idx+2), pointList[idx].Address)
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("D%d", idx+2), typedataOption[pointList[idx].TypeData])
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("E%d", idx+2), pointList[idx].Formula)
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("F%d", idx+2), pointList[idx].Unformula)
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("G%d", idx+2), uploadwayOption[pointList[idx].UploadWay])
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("H%d", idx+2), pointList[idx].UploadParm)
		// 			myExcel.SetCellValue("配置", fmt.Sprintf("I%d", idx+2), pointList[idx].UploadParm2)
		// 		}

		// 		myExcel.SetRowHeight("配置", 1, 20)
		// 		myExcel.SetColWidth("配置", "A", "A", 32)
		// 		myExcel.SetColWidth("配置", "B", "I", 20)

		// 		style, _ := myExcel.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}})
		// 		myExcel.SetCellStyle("配置", "A1", "I1", style)

		// 		style2, _ := myExcel.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Horizontal: "left"}})
		// 		myExcel.SetCellStyle("配置", "C2", fmt.Sprintf("C%d", len(pointList)+1), style2)
		// 		myExcel.SetCellStyle("配置", "H2", fmt.Sprintf("H%d", len(pointList)+1), style2)
		// 		myExcel.SetCellStyle("配置", "I2", fmt.Sprintf("I%d", len(pointList)+1), style2)

		// 		myExcel.SetCellValue("说明", "A1", "名称")
		// 		myExcel.SetCellValue("说明", "B1", "功能号")
		// 		myExcel.SetCellValue("说明", "C1", "通道地址")
		// 		myExcel.SetCellValue("说明", "D1", "数据类型")
		// 		myExcel.SetCellValue("说明", "E1", "处理公式")
		// 		myExcel.SetCellValue("说明", "F1", "逆处理公式")
		// 		myExcel.SetCellValue("说明", "G1", "上报模式")
		// 		myExcel.SetCellValue("说明", "H1", "上报参数一")
		// 		myExcel.SetCellValue("说明", "I1", "上报参数二")

		// 		myExcel.SetRowHeight("说明", 1, 20)
		// 		myExcel.SetColWidth("说明", "A", "A", 32)
		// 		myExcel.SetColWidth("说明", "B", "I", 20)

		// 		myExcel.SetCellStyle("说明", "A1", "I1", style)

		// 		myExcel.SetCellValue("说明", "A2", "变量的名称，不可重复")
		// 		for idx := range functionOption {
		// 			if idx > 0 {
		// 				myExcel.SetCellValue("说明", fmt.Sprintf("B%d", idx+1), functionOption[idx])
		// 			}
		// 		}
		// 		myExcel.SetCellValue("说明", "C2", "0 ~ 65534")
		// 		for idx := range typedataOption {
		// 			if idx > 0 {
		// 				myExcel.SetCellValue("说明", fmt.Sprintf("D%d", idx+1), typedataOption[idx])
		// 			}
		// 		}
		// 		myExcel.SetCellValue("说明", "E2", "例：(x + 15) / 6")
		// 		myExcel.SetCellValue("说明", "F2", "例：x * 6 - 15")
		// 		for idx := range uploadwayOption {
		// 			if idx > 0 {
		// 				myExcel.SetCellValue("说明", fmt.Sprintf("G%d", idx+1), uploadwayOption[idx])
		// 			}
		// 		}
		// 		for idx := range uploadParmOption {
		// 			myExcel.SetCellValue("说明", fmt.Sprintf("H%d", idx+2), uploadParmOption[idx])
		// 		}
		// 		for idx := range uploadParm2Option {
		// 			myExcel.SetCellValue("说明", fmt.Sprintf("I%d", idx+2), uploadParm2Option[idx])
		// 		}
		// 	}
		default:
			return "", errors.New("错误的通道类型")
		}
	}

	fileName = primitive.NewObjectID().Hex()
	err = myExcel.SaveAs("/var/www/dicloud/filecache/" + fileName + ".xlsx")

	return
}

// 导入
func (dao *ModuleTtdcDAO) ImportExcel(ctx *gin.Context, gmd model.GatewayModuleTtdc) (err error) {

	fileName := primitive.NewObjectID().Hex()

	moduleTem, res := dao.ExistModuleId(gmd)
	if !res {
		return errors.New("通道不存在")
	}

	if file, _, _err := ctx.Request.FormFile("file"); _err != nil {
		return errors.New("请求参数错误")
	} else {
		if out, _err := os.Create("./cache/" + fileName + ".xlsx"); _err != nil {
			return errors.New("请求参数错误")
		} else {
			if _, _err = io.Copy(out, file); _err != nil {
				return errors.New("请求参数错误")
			} else {
				/* - - ACTION - - */

				pointList := []moduleTtdcPointTransportDTO{}

				myExcel, _err := excelize.OpenFile("./cache/" + fileName + ".xlsx")
				if _err != nil {
					return errors.New("无效的文件")
				}
				defer os.Remove("./cache/" + fileName + ".xlsx")

				table, _err := myExcel.GetRows("配置")
				if _err != nil {
					return errors.New("无效的文件")
				}

				header := map[string]int{}
				for idx := range moduleTtdcHeaderOptions {
					header[idx] = moduleTtdcHeaderOptions[idx]
				}

				for idx := range table[0] {
					for _idx := range header {
						if _idx == table[0][idx] {
							header[_idx] = idx
						}
					}
				}

				for idx := range header {
					switch moduleTem.Type {
					case 1:
						switch idx {
						case "名称", "数据类型", "处理公式", "逆处理公式", "遥测量/遥信量",
							"字节序", "字节序号", "字节数", "是否取位", "取位位号":
							if header[idx] == -1 {
								return errors.New("表格缺少【" + idx + "】字段")
							}
						}

						// case 2, 3:
						// 	if header[idx] == -1 {
						// 		return errors.New("表格缺少【" + idx + "】字段")
						// 	}
					}
				}

				for row := range table {

					if table[row][header["名称"]] != "名称" {

						point := moduleTtdcPointTransportDTO{}

						if point.Name = table[row][header["名称"]]; point.Name == "" && row < len(table) {
							continue
						}
						if strings.Contains(point.Name, " ") {
							return fmt.Errorf("第 %d 行名称包含空格", row+1)
						}

						// 数据点ID - TODO: 或者导出导入更新
						point.DataId = fmt.Sprint(snowflakeutil.NextID())
						time.Sleep(1 * time.Millisecond)

						// switch moduleTem.Type {
						// case 2, 3:
						// 	if key := getOptionKey(functionOption, table[row][header["功能号"]]); key >= 0 {
						// 		point.Function = uint32(key)
						// 	} else {
						// 		return errors.New("数据【" + table[row][header["名称"]] + "】功能号错误")
						// 	}
						// 	if address, err := strconv.ParseInt(table[row][header["通道地址"]], 10, 32); err == nil {
						// 		point.Address = uint32(address)
						// 	} else {
						// 		return errors.New("数据【" + table[row][header["名称"]] + "】通道地址错误")
						// 	}
						// 	if key := getOptionKey(typedataOption, table[row][header["数据类型"]]); key >= 0 {
						// 		point.TypeData = uint32(key)
						// 	} else {
						// 		return errors.New("数据【" + table[row][header["名称"]] + "】数据类型错误")
						// 	}
						// }

						if point.Formula = table[row][header["处理公式"]]; point.Formula == "" {
							point.Formula = "x"
						}
						if point.Unformula = table[row][header["逆处理公式"]]; point.Unformula == "" {
							point.Unformula = "x"
						}

						// 数据再转换 - Excel数据转数据库数据
						if key := getModuleTtdcOptionKey(moduleTtdcDataTypeOptions, table[row][header["数据类型"]]); key >= 0 {
							point.DataType = key
						} else {
							return errors.New("数据【" + table[row][header["名称"]] + "】数据类型")
						}
						if key := getModuleTtdcOptionKey(moduleTtdcTelTypeOptions, table[row][header["遥测量/遥信量"]]); key >= 0 {
							point.TelType = uint32(key)
						} else {
							return errors.New("数据【" + table[row][header["名称"]] + "】遥测量/遥信量")
						}
						if key := getModuleTtdcOptionStrKey(moduleTtdcByteOrderOptions, table[row][header["字节序"]]); len(key) > 0 {
							point.ByteOrder = key
						} else {
							return errors.New("数据【" + table[row][header["名称"]] + "】字节序")
						}
						if key := getModuleTtdcOptionKey(moduleTtdcByteNumOptions, table[row][header["字节序号"]]); key >= 0 {
							point.ByteNum = uint32(key)
						} else {
							return errors.New("数据【" + table[row][header["名称"]] + "】字节序号")
						}
						if key := getModuleTtdcOptionKey(moduleTtdcByteCountOptions, table[row][header["字节数"]]); key >= 0 {
							point.ByteCount = uint32(key)
						} else {
							return errors.New("数据【" + table[row][header["名称"]] + "】字节数")
						}
						if key := getModuleTtdcOptionKey(moduleTtdcDoBitOptions, table[row][header["是否取位"]]); key >= 0 {
							point.DoBit = uint32(key)
						} else {
							return errors.New("数据【" + table[row][header["名称"]] + "】是否取位")
						}
						if key := getModuleTtdcOptionKey(moduleTtdcBitNumOptions, table[row][header["取位位号"]]); key >= 0 {
							point.BitNum = uint32(key)
						} else {
							return errors.New("数据【" + table[row][header["名称"]] + "】取位位号")
						}

						pointList = append(pointList, point)
					}
				}

				var gatewayDatas = []model.GatewayData{}
				for idx := range pointList {

					existId := -1
					for idx2 := range moduleTem.ModuleDatas {
						if pointList[idx].Name == moduleTem.ModuleDatas[idx2].Name {
							// return errors.New("名称【" + data.List[idx].Name + "】被已创建")
							existId = idx2
						}
					}

					if existId == -1 {
						gatewayDatas = append(gatewayDatas, model.GatewayData{
							GatewayId: moduleTem.GatewayId,
							Name:      pointList[idx].Name,
							DataId:    pointList[idx].DataId,
							DataType:  pointList[idx].DataType,
							Dbconfig:  -1,
						})
					}

					transport := ""
					switch moduleTem.Type {
					case 1:
						transport = fmt.Sprintf(
							"%d %s %d %d %d %d",
							pointList[idx].TelType, pointList[idx].ByteOrder, pointList[idx].ByteNum, pointList[idx].ByteCount,
							pointList[idx].DoBit, pointList[idx].BitNum,
						)
						// case 2, 3:
						// 	transport = fmt.Sprintf(
						// 		"%g %d %d %d %s %s %d %d",
						// 		pointList[idx].UploadParm, pointList[idx].Function, pointList[idx].TypeData, pointList[idx].Address,
						// 		pointList[idx].Formula, pointList[idx].Unformula,
						// 		pointList[idx].UploadWay, pointList[idx].UploadParm2,
						// 	)
					}

					if existId == -1 {
						moduleTem.ModuleDatas = append(moduleTem.ModuleDatas, model.ModuleTtdcData{
							ModuleId:  int(moduleTem.ID),
							Name:      pointList[idx].Name,
							DataType:  pointList[idx].DataType,
							Transport: transport,
							Formula:   pointList[idx].Formula,
							Unformula: pointList[idx].Unformula,
						})
					} else {
						moduleTem.ModuleDatas[existId].Transport = transport
					}
				}

				// 开启事务
				tx := dao.db.Begin()
				// 保存通道数据
				err := tx.Model(&model.ModuleTtdcData{}).Create(&moduleTem.ModuleDatas).Error
				if err != nil {
					log.Println("[ModuleTtdcDAO] ImportExcel create module datas fail")
					tx.Rollback()
					return err
				}

				// 保存网关数据
				err = tx.Model(&model.GatewayData{}).Create(&gatewayDatas).Error
				if err != nil {
					log.Println("[ModuleTtdcDAO] ImportExcel gateway datas fail")
					tx.Rollback()
					return err
				}

				// 提交事务
				tx.Commit()
			}
			defer out.Close()
		}
	}

	return
}

func getModuleTtdcOptionKey(optionMap map[int]string, valP string) int {

	for key, val := range optionMap {
		if val == valP {
			return key
		}
	}

	return -1
}

func getModuleTtdcOptionStrKey(optionMap map[string]string, valP string) string {

	for key, val := range optionMap {
		if val == valP {
			return key
		}
	}

	return ""
}
