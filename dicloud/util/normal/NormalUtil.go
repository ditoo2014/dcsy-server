package normalutil

import (
	"cmp"
	"fmt"
	"strconv"
)

func MinOf(vars ...int) int {
	min := vars[0]

	for _, i := range vars {
		if min > i {
			min = i
		}
	}

	return min
}

func MinOfInt(vars []string) int64 {
	// 10是基数，64表示返回int64类型
	min, err := strconv.ParseInt(vars[0], 10, 64)

	if err != nil {
		fmt.Println("[NormalUtil] MinOfInt", err)
		return 0
	}
	for _, i := range vars {
		i, err := strconv.ParseInt(i, 10, 64)
		if err != nil {
			fmt.Println("[NormalUtil] MinOfInt", err)
			return 0
		}
		if min > i {
			min = i
		}
	}

	return min
}

func MaxOfInt(vars []int) int {
	// 10是基数，64表示返回int64类型
	max := vars[0]

	for _, i := range vars {
		if max < i {
			max = i
		}
	}

	return max
}

func StringArrToIntArr(vars []string) (res []int64) {

	for _, i := range vars {
		if len(i) > 0 {
			i, err := strconv.ParseInt(i, 10, 64)
			if err != nil {
				fmt.Println("[NormalUtil] StringArrToIntArr", err)
				return
			}
			res = append(res, i)
		}
	}

	return
}

// 去重, go1.21+泛型
func Unique[T cmp.Ordered](ss []T) []T {
	size := len(ss)
	if size == 0 {
		return []T{}
	}

	// 这里新建一个切片, 大于为0, 因为我们不知道有几个非重复数据, 后面都使用append来动态增加并扩容
	newSlices := make([]T, 0)
	m1 := make(map[T]byte)
	for _, v := range ss {
		// 如果数据不在map中, 放入
		if _, ok := m1[v]; !ok {
			// 保存在map中, 用于下次判断
			m1[v] = 1
			// 将数据放入新的切片中
			newSlices = append(newSlices, v)
		}
	}

	return newSlices
}

// 获取分页数据 offset, limit
func GetPageOffsetLimit(page, pageSize int) (offset int, limit int) {

	// 默认页码
	if page <= 0 {
		page = 1
	}

	// 默认页大小
	switch {
	case pageSize > 1000:
		pageSize = 1000
	case pageSize <= 0:
		pageSize = 10
	}

	// 查询偏移量
	offset = (page - 1) * pageSize
	// 查询条数
	limit = pageSize
	return
}
