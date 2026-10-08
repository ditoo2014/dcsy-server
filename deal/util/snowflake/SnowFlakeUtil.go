package snowflakeutil

import (
	"log"
	"sync"
	"time"
)

const (
	// 定义各部分的位数
	epoch int64 = 1735660800000
	// 起始时间：2025-01-01 00:00:00 UTC
	timestampBits uint8 = 41
	machineIDBits uint8 = 10
	sequenceBits  uint8 = 12
	// 定义最大值
	maxMachineID int64 = -1 ^ (-1 << machineIDBits)
	maxSequence  int64 = -1 ^ (-1 << sequenceBits)
	// 定义位移
	timestampShift = machineIDBits + sequenceBits
	machineIDShift = sequenceBits
)

var (
	mutex     sync.Mutex
	lastStamp int64
	machineID int64 = 226
	sequence  int64
)

// NextID 生成下一个唯一ID
func NextID() int64 {
	mutex.Lock()
	defer mutex.Unlock()
	// 获取当前时间戳
	currentStamp := time.Now().UnixMilli()
	// 如果当前时间戳小于上次生成ID的时间戳，说明时钟回拨
	if currentStamp < lastStamp {
		log.Printf("[SNOWFLAKE] 时钟回拨, 等待直到 %d\n", lastStamp)
		// 等待直到时钟追上上次生成ID的时间戳
		for currentStamp <= lastStamp {
			time.Sleep(100 * time.Microsecond) // 等待100微秒后重试
			currentStamp = time.Now().UnixMilli()
		}
		log.Printf("[SNOWFLAKE] 时钟追上 %d\n", currentStamp)
	}
	// 如果是同一毫秒内生成的ID，则递增序列号
	if currentStamp == lastStamp {
		sequence = (sequence + 1) & maxSequence
		// 如果序列号溢出，则等待下一毫秒
		if sequence == 0 {
			for currentStamp <= lastStamp {
				currentStamp = time.Now().UnixMilli()
			}
		}
	} else {
		// 如果不是同一毫秒，则重置序列号
		sequence = 0
	}
	// 更新上次生成ID的时间戳
	lastStamp = currentStamp
	// 生成ID
	id := ((currentStamp - epoch) << timestampShift) | (machineID << machineIDShift) | sequence
	return id
}
