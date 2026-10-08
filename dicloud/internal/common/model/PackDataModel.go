package model

import (
	"errors"
	"fmt"
	"time"
)

/* dtdp-old */

type PackDtdpOldData struct {
	Port    int64       `bson:"port" json:"port"`
	DevId   int64       `bson:"devId" json:"devId"`
	Proto   int64       `bson:"proto" json:"proto"`
	Func    int64       `bson:"func" json:"func"`
	Payload []DtdpOldDl `bson:"payload" json:"payload"`
}
type DtdpOldDl struct {
	Len int64    `bson:"len" json:"len"`
	Adr int64    `bson:"adr" json:"adr"`
	Val []uint16 `bson:"val" json:"val"`
}

/* dtdp */

type PackDtdpData struct {
	T      int64    `bson:"t" json:"t"`
	Dl     []DtdpDl `bson:"dl" json:"dl"`
	DbName string   `json:"dbName"` // multi renter - 多租户
}
type DtdpDl struct {
	ID string  `bson:"id" json:"id"`
	S  int     `bson:"s" json:"s"`
	V  float64 `bson:"v" json:"v"`
}

/* dtdp str */

type PackDtdpDataStr struct {
	T      int64       `bson:"t" json:"t"`
	Dl     []DtdpDlStr `bson:"dl" json:"dl"`
	DbName string      `json:"dbName"` // multi renter - 多租户
}
type DtdpDlStr struct {
	ID string `bson:"id" json:"id"`
	S  int    `bson:"s" json:"s"`
	V  string `bson:"v" json:"v"`
}

/* mcdp */

// 上报 - 数据
type PackMcdpData struct {
	B McdpDvl `bson:"b" json:"b"`
}
type McdpDvl struct {
	Ts  string   `bson:"ts" json:"ts"`
	Dvl []McdpDl `bson:"dvl" json:"dvl"`
}
type McdpDl struct {
	Da string                 `bson:"da" json:"da"`
	Dl map[string]interface{} `bson:"dl" json:"dl"`
}

// 下发 - 写操作
type PackMccpCmdWrite struct {
	H MccpCmdWriteH `bson:"h" json:"h"`
	B MccpCmdWriteB `bson:"b" json:"b"`
}
type MccpCmdWriteH struct {
	Rt string `bson:"rt" json:"rt"`
}
type MccpCmdWriteB struct {
	Dl map[string]string `bson:"dl" json:"dl"`
}

/* zydp */

type PackZydpData struct {
	Increment int       `bson:"increment" json:"increment"`
	Cmd       int       `bson:"cmd" json:"cmd"`
	Power     int       `bson:"power" json:"power"`
	Signal    int       `bson:"signal" json:"signal"`
	IMEI      string    `bson:"imei" json:"imei"`
	Data      []float32 `bson:"data" json:"data"`
}

func (p *PackZydpData) Parsing(msg []byte) (err error) {

	// 校验帧头
	if 0xAA != msg[0] || 0x55 != msg[1] {
		err = errors.New("帧头错误")
		return
	}

	// 校验和
	// var check int

	// for idx := 0; idx < len(msg)-1; idx++ {
	// 	check += int(msg[idx])
	// }

	// fmt.Println(check)

	// check %= 256

	// fmt.Println(check)

	// if check != int(msg[len(msg)-1]) {
	// 	err = errors.New("和校验错误")
	// 	return
	// }

	// 赋值
	p.Increment = int(msg[2])
	p.Cmd = int(msg[3])
	p.Power = int(msg[4])
	p.Signal = int(msg[5])/16*10 + int(msg[5])%16
	p.IMEI = fmt.Sprintf(
		"%02x%02x%02x%02x%02x%02x%02x%02x",
		msg[6], msg[7], msg[8], msg[9],
		msg[10], msg[11], msg[12], msg[13],
	)

	switch p.Cmd {
	case 0:
	case 1:
	case 2:
	case 3:
		for idx := 15; idx < 15+int(msg[14]); idx++ {
			if 128 < int(msg[idx]) {
				p.Data = append(p.Data, -float32(msg[idx]))
			} else {
				p.Data = append(p.Data, float32(msg[idx]))
			}
		}
	case 4:
		for idx := 15; idx < 15+int(msg[14]); idx++ {
			x1 := float32(int(msg[idx]) / 16)
			x2 := float32(int(msg[idx]) % 16)

			for x2 > 1 {
				x2 /= 10
			}

			p.Data = append(p.Data, x1+x2)
		}
	default:
	}

	return
}

// 通用保存数据
type DealData struct {
	time  time.Time
	speed int
}
