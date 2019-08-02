package common

import (
	"superserver/until/common/mersenneTwister"
	"math"
	"sync"
	"time"
)

//GlobalRand 全局随机数发生器
var GlobalRand = mersenneTwister.New()

func init() {
	cardOnce := &sync.Once{}
	cardOnce.Do(func() {
		// rand.Seed(time.Now().UnixNano())
		GlobalRand.Seed(time.Now().UnixNano())
	})
}

//ShuffleFisherYatesI 接口
type ShuffleFisherYatesI interface {
	Swap(a int, b int)
	Len() int
}

//ShuffleFisherYates shuffler arrays with  Fisher_Yates
//https://en.wikipedia.org/wiki/Fisher-Yates_shuffle
func ShuffleFisherYates(elements ShuffleFisherYatesI) {
	i := elements.Len()
	if i == 0 {
		return
	}
	for {
		i--
		if i == 0 {
			break
		}
		// j := rand.Intn(i + 1)
		j := GlobalRand.IntN(uint64(i + 1))
		elements.Swap(i, int(j))
	}
}

// EarthDistance 计算经纬度的球面距离，返回值的单位为米
//参数顺序为：纬度1，经度1，纬度2，经度2
func EarthDistance(lat1, lng1, lat2, lng2 float64) float64 {
	radius := float64(6371000) // 6378137
	rad := math.Pi / 180.0
	lat1 = lat1 * rad
	lng1 = lng1 * rad
	lat2 = lat2 * rad
	lng2 = lng2 * rad
	theta := lng2 - lng1
	dist := math.Acos(math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(theta))
	return dist * radius
}
