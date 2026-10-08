// 处理 定时触发器
package service

import (
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

func DealEvent(msgOrg *primitive.MessageExt) {

	// 获取数据, 转换数据类型
	// msg := string(msgOrg.Body)

	// array := strings.Split(msg, " ")

	// if 3 != len(array) {
	// 	log.Println("[DealEvent] len != 3", nil)
	// 	return
	// }

	// begin, err := strconv.ParseInt(array[0], 10, 64)
	// if err != nil {
	// 	log.Println("[DealEvent] dealEvent strconv.ParseInt", err)
	// 	return
	// }

	// LabelID, err := primitive.ObjectIDFromHex(array[1])
	// if err != nil {
	// 	log.Println("[DealEvent] dealEvent ObjectIDFromHex array[1]", err)
	// 	return
	// }

	// timingID, _err := primitive.ObjectIDFromHex(array[2])
	// if _err != nil {
	// 	log.Println("[DealEvent] dealEvent ObjectIDFromHex array[2]", err)
	// 	return
	// }

	// _dealTrigger(begin, LabelID, timingID)
}

// func _dealTrigger(begin int64, LabelID primitive.ObjectID, timingID primitive.ObjectID) (err error) {

// timing := trigger.Timing{ID: timingID}

// timing, err = timing.Exist()
// if nil != err {
// 	log.Println("[DealEvent] _dealTrigger Exist", err)
// 	return
// }

// if 0 == len(timing.Mapping) {
// 	return
// }
// if 0 == len(timing.Datacome) {
// 	return
// }
// if 0 == len(timing.Logic) {
// 	return
// }
// if 0 == len(timing.Topic) {
// 	return
// }

// if timing.Cycle <= begin-timing.Timer {

// 	stream := model.Stream{Label: LabelID}

// 	fmsg := ""

// 	values, _err := stream.GetDataFormula(timing.Mapping, timing.Datacome)
// 	if nil != _err {
// 		fmsg = "数据提取错误"
// 		return
// 	}

// 	if len(values) != len(timing.Datacome) {
// 		log.Println("[DealEvent] _dealTrigger len values != Datacome", nil)
// 		return
// 	}

// 	for idx := range timing.Datacome {
// 		value, exist := values[timing.Datacome[idx]]
// 		if exist {
// 			timing.Datacome[idx] = strconv.FormatFloat(value, 'f', 3, 64)
// 		}
// 	}

// 	value, dmsg := timing.GetLogic()

// 	result, rmsg := timing.GetResault()

// 	pack := trigger.PackTiming{
// 		Coding: timing.Coding,
// 		Begin:  begin,
// 		End:    time.Now().Unix(),
// 		Value:  value,
// 		Result: result,
// 		Err:    fmsg + dmsg + rmsg,
// 	}

// 	_str, _err := pack.ToJsonStr()

// 	mqtt := common.GetMqtt()
// 	mqtt.Pub(timing.Topic, _str)

// 	err = timing.SetTimer(-1, begin)
// 	if nil != err {
// 		log.Println("[DealEvent] _dealTrigger SetTimer", err)
// 		return
// 	}
// }

// return
// }
