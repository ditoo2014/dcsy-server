package service

import (
	"dicloud/internal/tolink/model"
	"dicloud/util/opengauss"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type TtdcAlarmRulePageRequest struct {
	PageNum  int    `json:"pageNum" form:"pageNum"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	GroupID  uint   `json:"groupId" form:"groupId"`
	Keyword  string `json:"keyword" form:"keyword"`
}

type TtdcAlarmRuleLogicRequest struct {
	ID           uint    `json:"id"`
	Logic        string  `json:"logic"`
	TriggerValue float64 `json:"triggerValue"`
	MatchValue   string  `json:"matchValue"`
	SortNo       int     `json:"sortNo"`
}

type TtdcAlarmRuleRequest struct {
	ID              uint                        `json:"id"`
	TemplateGroupID uint                        `json:"templateGroupId"`
	TemplateID      uint                        `json:"templateId"`
	TemplatePointID uint                        `json:"templatePointId"`
	BaseDataID      string                      `json:"baseDataId"`
	RuleName        string                      `json:"ruleName"`
	RuleCode        string                      `json:"ruleCode"`
	AlarmType       string                      `json:"alarmType"`
	AlarmStage      int                         `json:"alarmStage"`
	AlarmChannel    string                      `json:"alarmChannel"`
	AlarmUsers      string                      `json:"alarmUsers"`
	TriggerModel    int                         `json:"triggerModel"`
	Formula         string                      `json:"formula"`
	Cycle           int64                       `json:"cycle"`
	SustainTime     int64                       `json:"sustainTime"`
	Enabled         *bool                       `json:"enabled"`
	Version         int                         `json:"version"`
	SortNo          int                         `json:"sortNo"`
	Description     string                      `json:"description"`
	Logics          []TtdcAlarmRuleLogicRequest `json:"logics"`
}

type TtdcAlarmRuleService struct{ db *gorm.DB }

func NewTtdcAlarmRuleService() *TtdcAlarmRuleService {
	return &TtdcAlarmRuleService{db: opengauss.GetDefaultGormDB()}
}

func (s *TtdcAlarmRuleService) checkDB() error {
	if s == nil || s.db == nil {
		return errors.New("默认OpenGauss数据源未初始化")
	}
	return requireTtdcSchema(s.db, false)
}

func (s *TtdcAlarmRuleService) FindPage(req TtdcAlarmRulePageRequest) ([]model.TtdcAlarmRule, int64, error) {
	if err := s.checkDB(); err != nil {
		return nil, 0, err
	}
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 || req.PageSize > 200 {
		req.PageSize = 20
	}
	query := s.db.Model(&model.TtdcAlarmRule{}).Order("id asc")
	if req.GroupID > 0 {
		query = query.Where("template_group_id = ?", req.GroupID)
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("rule_name LIKE ? OR rule_code LIKE ? OR base_data_id LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.TtdcAlarmRule
	if err := query.Preload("Logics", func(db *gorm.DB) *gorm.DB { return db.Order("sort_no asc, id asc") }).Offset((req.PageNum - 1) * req.PageSize).Limit(req.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	if err := s.attachPointNames(rows); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *TtdcAlarmRuleService) Detail(id uint) (model.TtdcAlarmRule, error) {
	var item model.TtdcAlarmRule
	if err := s.checkDB(); err != nil {
		return item, err
	}
	if err := s.db.Preload("Logics", func(db *gorm.DB) *gorm.DB { return db.Order("sort_no asc, id asc") }).First(&item, id).Error; err != nil {
		return item, err
	}
	rows := []model.TtdcAlarmRule{item}
	if err := s.attachPointNames(rows); err != nil {
		return item, err
	}
	item.TemplateName, item.PointName = rows[0].TemplateName, rows[0].PointName
	return item, nil
}

func (s *TtdcAlarmRuleService) attachPointNames(rows []model.TtdcAlarmRule) error {
	for index := range rows {
		var template model.TtdcTemplate
		if err := s.db.Select("id, template_name").First(&template, rows[index].TemplateID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		rows[index].TemplateName = template.TemplateName
		var point model.TtdcTemplatePoint
		if err := s.db.Select("id, name").First(&point, rows[index].TemplatePointID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		rows[index].PointName = point.Name
	}
	return nil
}

func (s *TtdcAlarmRuleService) Save(req TtdcAlarmRuleRequest, userID int) (model.TtdcAlarmRule, error) {
	item, logics, err := normalizeTtdcAlarmRuleRequest(req)
	if err != nil {
		return item, err
	}
	if err = s.checkDB(); err != nil {
		return item, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var group model.TtdcTemplateGroup
		if err := tx.First(&group, item.TemplateGroupID).Error; err != nil {
			return errors.New("模板组不存在")
		}
		var template model.TtdcTemplate
		if err := tx.Where("id = ? AND group_id = ? AND deleted_at IS NULL", item.TemplateID, item.TemplateGroupID).First(&template).Error; err != nil {
			return errors.New("告警规则对应的通道模板不存在")
		}
		var point model.TtdcTemplatePoint
		if err := tx.Where("id = ? AND template_id = ? AND group_id = ? AND deleted_at IS NULL", item.TemplatePointID, item.TemplateID, item.TemplateGroupID).First(&point).Error; err != nil {
			return errors.New("告警规则对应的数据点不存在")
		}
		if item.BaseDataID != point.BaseDataID {
			return errors.New("告警规则数据点已失效，请重新选择数据点")
		}
		if point.DataType != 1 {
			return errors.New("告警规则只能绑定数字数据点")
		}
		if item.ID == 0 {
			var count int64
			if err := tx.Where("rule_code = ?", item.RuleCode).Model(&model.TtdcAlarmRule{}).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("公共告警代码已存在")
			}
			item.CreatedBy, item.UpdatedBy = userID, userID
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		} else {
			var current model.TtdcAlarmRule
			if err := tx.First(&current, item.ID).Error; err != nil {
				return err
			}
			if req.Version > 0 && req.Version != current.Version {
				return errors.New("告警规则已被其他用户修改")
			}
			var count int64
			if err := tx.Where("rule_code = ? AND id <> ?", item.RuleCode, item.ID).Model(&model.TtdcAlarmRule{}).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("公共告警代码已存在")
			}
			item.CreatedBy = current.CreatedBy
			item.Version, item.UpdatedBy = current.Version+1, userID
			if err := tx.Model(&current).Select("template_group_id", "template_id", "template_point_id", "base_data_id", "rule_name", "rule_code", "alarm_type", "alarm_stage", "alarm_channel", "alarm_users", "trigger_model", "formula", "cycle", "sustain_time", "enabled", "version", "sort_no", "description", "updated_by").Updates(&item).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("rule_id = ?", item.ID).Delete(&model.TtdcAlarmRuleLogic{}).Error; err != nil {
			return err
		}
		for index := range logics {
			logics[index].RuleID = item.ID
			if err := tx.Create(&logics[index]).Error; err != nil {
				return err
			}
		}
		operation := "create"
		if req.ID > 0 {
			operation = "update"
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "alarm_rule", TargetKey: strconv.FormatUint(uint64(item.ID), 10), Operation: operation}).Error
	})
	if err != nil {
		return item, err
	}
	return s.Detail(item.ID)
}

func (s *TtdcAlarmRuleService) SetEnabled(id uint, value bool) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.TtdcAlarmRule{}).Where("id = ?", id).Update("enabled", value).Error; err != nil {
			return err
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "alarm_rule", TargetKey: strconv.FormatUint(uint64(id), 10), Operation: "update"}).Error
	})
}

func (s *TtdcAlarmRuleService) Delete(id uint) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var item model.TtdcAlarmRule
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}
		if err := tx.Where("rule_id = ?", id).Delete(&model.TtdcAlarmRuleLogic{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&item).Error; err != nil {
			return err
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "alarm_rule", TargetKey: strconv.FormatUint(uint64(id), 10), Operation: "delete"}).Error
	})
}

func normalizeTtdcAlarmRuleRequest(req TtdcAlarmRuleRequest) (model.TtdcAlarmRule, []model.TtdcAlarmRuleLogic, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item := model.TtdcAlarmRule{
		ID: req.ID, TemplateGroupID: req.TemplateGroupID, TemplateID: req.TemplateID, TemplatePointID: req.TemplatePointID,
		BaseDataID: strings.TrimSpace(req.BaseDataID), RuleName: strings.TrimSpace(req.RuleName), RuleCode: strings.TrimSpace(req.RuleCode),
		AlarmType: strings.TrimSpace(req.AlarmType), AlarmStage: req.AlarmStage,
		// Public TTDC rules are department-queryable records. They never carry
		// recipients, channels, or a user-configurable rule order.
		AlarmChannel: "", AlarmUsers: "", TriggerModel: 0, Formula: strings.TrimSpace(req.Formula),
		// Public TTDC alarm rules always use the single-trigger mode. The
		// legacy cycle field remains in the schema for compatibility only.
		Cycle: 0, SustainTime: req.SustainTime, Enabled: enabled, Version: req.Version, SortNo: 0, Description: strings.TrimSpace(req.Description),
	}
	if item.TemplateGroupID == 0 || item.TemplateID == 0 || item.TemplatePointID == 0 {
		return item, nil, errors.New("请选择模板组、通道和数据点")
	}
	if item.RuleName == "" || len(item.RuleName) > 128 || item.RuleCode == "" || len(item.RuleCode) > 128 {
		return item, nil, errors.New("规则名称和告警代码不能为空且不能超过128个字符")
	}
	if item.AlarmType != "0" && item.AlarmType != "1" {
		return item, nil, errors.New("告警类型只能是0告警或1预警")
	}
	if item.AlarmStage <= 0 {
		item.AlarmStage = 1
	}
	if item.AlarmStage > 3 {
		return item, nil, errors.New("告警等级只能是一级、二级或三级")
	}
	if item.Formula == "" {
		item.Formula = "x"
	}
	if item.Version <= 0 {
		item.Version = 1
	}
	if item.SustainTime < 0 {
		return item, nil, errors.New("持续时间不能为负数")
	}
	if len(req.Logics) == 0 {
		return item, nil, errors.New("至少配置一条告警判断条件")
	}
	logics := make([]model.TtdcAlarmRuleLogic, 0, len(req.Logics))
	allowed := map[string]bool{">": true, ">=": true, "<": true, "<=": true, "=": true}
	for index, logic := range req.Logics {
		operator := strings.TrimSpace(logic.Logic)
		if !allowed[operator] {
			return item, nil, errors.New("判断条件的运算符不合法")
		}
		logics = append(logics, model.TtdcAlarmRuleLogic{ID: logic.ID, Logic: operator, TriggerValue: logic.TriggerValue, SortNo: index + 1})
	}
	return item, logics, nil
}
