// 产生事件 实时触发器
package service

var EventTopic string

// var threand int64

// var begin int64

// func dealEvent() {

// begin = time.Now().Unix()

// unit := trigger.TimingConfig{Marktrigger: "timing  common"}

// acount, err := unit.GetAcount()
// if nil != err {
// 	log.Println("dealEvent GetAcount", err)
// 	return
// }

// chanel := make(chan int64, threand)

// var idx int64
// for idx = 1; idx < acount/threand+1+1; idx++ {
// 	chanel <- idx - 1

// 	page := model.Page{
// 		Pagenum:  idx,
// 		Pagesize: threand,
// 	}

// 	go _dealTriggerList(&chanel, &unit, page)

// }
// }

// func _dealTriggerList(chanel *chan int64, unit *trigger.TimingConfig, page model.Page) (err error) {

// 	list, _err := unit.FindPage(page)
// 	if nil != _err {
// 		goto END
// 	}

// 	for idx := range list {
// 		_str := fmt.Sprintf("%d %s %s", begin, list[idx].Config.Hex(), list[idx].Trigger.Hex())
// 		nsq := common.GetNsq()
// 		nsq.Pub(EventTopic, _str)
// 	}

// END:
// 	<-*chanel
// 	return
// }
