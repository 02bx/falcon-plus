package proc

import (
	P "github.com/open-falcon/common/proc"
	"log"
	"time"
)

// 索引更新
var (
	IndexUpdateAll = P.NewSCounterQps("IndexUpdateAll")
	IndexDelete    = P.NewSCounterQps("IndexDelete")
	IndexDeleteCnt = P.NewSCounterBase("IndexDeleteCnt")
)

// transfer监控数据采集
var (
	MonitorCronCnt = P.NewSCounterQps("MonitorCronCnt")
)

func Init() {
	log.Println("proc:Init, ok")
}

func GetAll() []interface{} {
	ret := make([]interface{}, 0)

	// index
	ret = append(ret, IndexUpdateAll.Get())
	ret = append(ret, IndexDelete.Get())
	ret = append(ret, IndexDeleteCnt.Get())

	// monitor
	ret = append(ret, MonitorCronCnt.Get())

	return ret
}

// TODO 临时放在这里了, 考虑放到合适的模块
func FmtUnixTs(ts int64) string {
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}
