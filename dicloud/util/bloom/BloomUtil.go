package bloomutil

import "github.com/bits-and-blooms/bloom"

var LocalBloomFilter *bloom.BloomFilter
var BusinessGeneralTriggerBloomFilter *bloom.BloomFilter

// 预计要插入的元素数量, 一百万个数据点, 超过另做处理

var expectedItems = 1000000

// 可接受的误判率

var falsePositiveRate = 0.01

func InitialBloomFilter() {

	// 创建布隆过滤器

	LocalBloomFilter = bloom.NewWithEstimates(uint(expectedItems), falsePositiveRate)
	BusinessGeneralTriggerBloomFilter = bloom.NewWithEstimates(uint(expectedItems), falsePositiveRate)
}
