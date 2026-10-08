package service

import (
	"dicloud/internal/business/client"
	"dicloud/internal/business/dao"
	businessmodel "dicloud/internal/business/model"
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/common/model"
	tolinkdao "dicloud/internal/tolink/dao"
	tolinkmodel "dicloud/internal/tolink/model"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetGeneralMarktriggerListByLabelName(ctx *gin.Context, rangeEquipmentId []int64, labelName string) (gmts []model.GeneralMarktrigger) {
	datadetailDao := dao.NewDataDetailsDao(ctx)
	dds, res := datadetailDao.FindDataDetailsByLabelNameOrName(labelName)
	if !res {
		log.Println("[BUSINESS] GeneralTrigger get marktrigger labelName bind equipment is nil")
		return
	}

	equipmentIds := []int{}
	dataDetailsByEquipmentId := map[int]businessmodel.DataDetails{}
	for _, vTem := range dds {
		if vTem.EquipmentId == 0 {
			continue
		}
		if len(rangeEquipmentId) > 0 && !generalTriggerEquipmentInRange(rangeEquipmentId, vTem.EquipmentId) {
			continue
		}
		if _, exist := dataDetailsByEquipmentId[vTem.EquipmentId]; !exist {
			equipmentIds = append(equipmentIds, vTem.EquipmentId)
		}
		if shouldReplaceGeneralTriggerDataDetail(dataDetailsByEquipmentId[vTem.EquipmentId], vTem, labelName) {
			dataDetailsByEquipmentId[vTem.EquipmentId] = vTem
		}
	}

	equipmentDao := dao.NewEquipmentDAO(ctx)
	equipmentList, res := equipmentDao.FindEquipmentByIdArr(businessmodel.Equipment{}, equipmentIds)
	if !res {
		log.Println("[BUSINESS] GeneralTrigger get marktrigger equipment is nil")
		return
	}

	for _, vTem := range equipmentList {
		if len(vTem.ApiKey) == 0 {
			log.Println("[BUSINESS] GeneralTrigger get marktrigger apiKey is nil")
			continue
		}

		resp, err := client.ExistApikey(ctx, &tolinkmodel.Stream{ApiKey: vTem.ApiKey})
		if err != nil || resp == nil || resp.Code != 200 || resp.Stream == nil {
			log.Println("[BUSINESS] GeneralTrigger call gRPC fail, err: ", err)
			continue
		}

		dataDetail, ok := dataDetailsByEquipmentId[int(vTem.ID)]
		if !ok {
			log.Println("[BUSINESS] GeneralTrigger get marktrigger data detail cannot find, equipmentId: ", vTem.ID)
			continue
		}
		streamLabelName := strings.TrimSpace(dataDetail.LabelName)
		if streamLabelName == "" {
			log.Println("[BUSINESS] GeneralTrigger get marktrigger data detail labelName is nil, equipmentId: ", vTem.ID)
			continue
		}

		isExist := -1
		labelID := strings.TrimSpace(dataDetail.LabelId)
		for idx, labelTem := range resp.Stream.List {
			if labelTem != nil && labelID != "" && strings.TrimSpace(labelTem.LabelId) == labelID {
				isExist = idx
				break
			}
			if labelTem != nil && isExist == -1 && strings.TrimSpace(labelTem.Name) == streamLabelName {
				isExist = idx
			}
		}
		if isExist == -1 {
			log.Println("[BUSINESS] GeneralTrigger get marktrigger stream label cannot find")
			continue
		}

		marktriggerTem := model.GeneralMarktrigger{}
		marktriggerTem.Marktrigger = resp.Stream.List[isExist].GatewayId + " " + resp.Stream.List[isExist].DataId
		marktriggerTem.SiteId = vTem.SiteId
		marktriggerTem.EquipmentId = int(vTem.ID)

		gmts = append(gmts, marktriggerTem)
	}

	return
}

// GetEquipmentHealthMarktriggerList resolves every SOH point exposed by the
// data pool bound to each equipment. Health statistics cannot rely on the
// business battery code or on one fixed data ID because the actual point may
// be alg_soh or a channel-prefixed value such as 01_alg_soh.
func GetEquipmentHealthMarktriggerList(ctx *gin.Context, rangeEquipmentID []int64) (gmts []model.GeneralMarktrigger) {
	if len(rangeEquipmentID) == 0 {
		return
	}

	equipmentIDs := make([]int, 0, len(rangeEquipmentID))
	requested := make(map[int]struct{}, len(rangeEquipmentID))
	for _, equipmentID := range rangeEquipmentID {
		if equipmentID <= 0 {
			continue
		}
		id := int(equipmentID)
		if _, exists := requested[id]; exists {
			continue
		}
		requested[id] = struct{}{}
		equipmentIDs = append(equipmentIDs, id)
	}
	if len(equipmentIDs) == 0 {
		return
	}

	equipmentDAO := dao.NewEquipmentDAO(ctx)
	equipmentList, ok := equipmentDAO.FindEquipmentByIdArr(businessmodel.Equipment{}, equipmentIDs)
	if !ok {
		log.Println("[BUSINESS] EquipmentHealth get equipment is nil")
		return
	}

	seen := make(map[string]struct{})
	for _, equipment := range equipmentList {
		if strings.TrimSpace(equipment.ApiKey) == "" {
			log.Println("[BUSINESS] EquipmentHealth get apiKey is nil, equipmentId: ", equipment.ID)
			continue
		}

		stream, ok := findEquipmentHealthStream(ctx, equipment.ApiKey)
		if !ok {
			log.Println("[BUSINESS] EquipmentHealth get stream fail, equipmentId: ", equipment.ID)
			continue
		}

		for _, label := range stream.List {
			if label == nil || strings.TrimSpace(label.GatewayId) == "" || !isEquipmentHealthStreamLabel(label) {
				continue
			}
			dataID := strings.TrimSpace(label.DataId)
			gatewayID := strings.TrimSpace(label.GatewayId)
			if dataID == "" {
				continue
			}
			key := fmt.Sprintf("%d\x00%s\x00%s", equipment.ID, gatewayID, dataID)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			gmts = append(gmts, model.GeneralMarktrigger{
				SiteId:      equipment.SiteId,
				EquipmentId: int(equipment.ID),
				Marktrigger: gatewayID + " " + dataID,
			})
		}
	}

	return
}

func findEquipmentHealthStream(ctx *gin.Context, apiKey string) (*tolink_grpc.Stream, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, false
	}

	// Resolve against the same tenant database used by the health-status page.
	// This avoids depending on a separate gRPC process (and its tenant context)
	// together with the realtime Redis read below.
	streamDAO := tolinkdao.NewStreamDAO(ctx)
	stream, exists := streamDAO.ExistApiKey(tolinkmodel.Stream{ApiKey: apiKey})
	if exists && stream.ID != 0 && streamHasEquipmentHealthPoint(stream.List) {
		return tolinkStreamGRPCData(stream), true
	}

	// Retain the gRPC fallback for deployments where business and tolink use
	// separate databases or the local stream tables are not present.
	resp, err := client.ExistApikey(ctx, &tolinkmodel.Stream{ApiKey: apiKey})
	if err != nil {
		log.Println("[BUSINESS] EquipmentHealth gRPC stream lookup failed: ", err)
	}
	if err == nil && resp != nil && resp.Code == 200 && resp.Stream != nil && streamHasEquipmentHealthPointGRPC(resp.Stream.List) {
		return resp.Stream, true
	}
	return nil, false
}

func streamHasEquipmentHealthPoint(labels []tolinkmodel.StreamLabelUnit) bool {
	for index := range labels {
		if isEquipmentHealthDataIDOrName(labels[index].DataId, labels[index].Name, labels[index].DataName) {
			return true
		}
	}
	return false
}

func streamHasEquipmentHealthPointGRPC(labels []*tolink_grpc.StreamLabelUnit) bool {
	for _, label := range labels {
		if label != nil && isEquipmentHealthDataIDOrName(label.DataId, label.Name, label.DataName) {
			return true
		}
	}
	return false
}

func isEquipmentHealthDataIDOrName(dataID, name, dataName string) bool {
	if isEquipmentHealthSOHDataID(dataID) {
		return true
	}
	return strings.Contains(strings.ToUpper(strings.TrimSpace(name)), "SOH") ||
		strings.Contains(strings.ToUpper(strings.TrimSpace(dataName)), "SOH")
}

func tolinkStreamGRPCData(stream tolinkmodel.Stream) *tolink_grpc.Stream {
	result := &tolink_grpc.Stream{
		Id:           int64(stream.ID),
		Admin:        stream.Admin,
		Name:         stream.Name,
		ApiKey:       stream.ApiKey,
		CreateUserId: int64(stream.CreateUserId),
		CorpId:       int64(stream.CorpId),
		StreamCode:   stream.StreamCode,
	}
	for index := range stream.List {
		label := stream.List[index]
		result.List = append(result.List, &tolink_grpc.StreamLabelUnit{
			StreamId:    int64(label.StreamId),
			LabelId:     label.LabelId,
			Name:        label.Name,
			GatewayName: label.GatewayName,
			GatewayId:   label.GatewayId,
			DataName:    label.DataName,
			DataType:    label.DataType,
			DataId:      label.DataId,
			Formula:     label.Formula,
			Unformula:   label.Unformula,
		})
	}
	return result
}

func isEquipmentHealthStreamLabel(label *tolink_grpc.StreamLabelUnit) bool {
	if label == nil {
		return false
	}
	return isEquipmentHealthDataIDOrName(label.DataId, label.Name, label.DataName)
}

func isEquipmentHealthSOHDataID(dataID string) bool {
	dataID = strings.ToLower(strings.TrimSpace(dataID))
	return dataID == "alg_soh" || strings.HasSuffix(dataID, "_alg_soh")
}

func generalTriggerEquipmentInRange(rangeEquipmentId []int64, equipmentId int) bool {
	for _, idTem := range rangeEquipmentId {
		if idTem == int64(equipmentId) {
			return true
		}
	}
	return false
}

func shouldReplaceGeneralTriggerDataDetail(current, next businessmodel.DataDetails, labelName string) bool {
	if current.ID == 0 {
		return true
	}

	labelName = strings.TrimSpace(labelName)
	currentExact := strings.TrimSpace(current.LabelName) == labelName
	nextExact := strings.TrimSpace(next.LabelName) == labelName

	return nextExact && !currentExact
}
