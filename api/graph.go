package api

import (
	"fmt"
	"log"
	"math"

	"github.com/open-falcon/common/model"
	MUtils "github.com/open-falcon/common/utils"
	"github.com/open-falcon/graph/g"
	"github.com/open-falcon/graph/index"
	"github.com/open-falcon/graph/proc"
	"github.com/open-falcon/graph/rrdtool"
	"github.com/open-falcon/graph/store"
	//"sync/atomic"
)

//var DropCounter int64

type Graph int

func (this *Graph) Ping(req model.NullRpcRequest, resp *model.SimpleRpcResponse) error {
	return nil
}

func (this *Graph) Send(items []*model.GraphItem, resp *model.SimpleRpcResponse) error {
	go handleItems(items)
	return nil
}

// 供外部调用、处理接收到的数据 的接口
func HandleItems(items []*model.GraphItem) error {
	handleItems(items)
	return nil
}

func handleItems(items []*model.GraphItem) {
	if items == nil {
		return
	}

	count := len(items)
	if count == 0 {
		return
	}

	for i := 0; i < count; i++ {
		if items[i] == nil {
			continue
		}
		checksum := items[i].Checksum()

		//statistics
		proc.GraphRpcRecvCnt.Incr()
		if checksum == proc.RecvDataTrace.PK {
			proc.RecvDataTrace.PushFront(items[i])
		}

		// To Graph
		first := store.GraphItems.First(checksum)
		if first != nil && items[i].Timestamp <= first.Timestamp {
			continue
		}
		store.GraphItems.PushFront(checksum, items[i])

		// To Index
		index.ReceiveItem(items[i], checksum)
	}
}

func (this *Graph) Query(param model.GraphQueryParam, resp *model.GraphQueryResponse) error {
	resp.Values = []*model.RRDData{}

	endpointId, exists := store.LoadEndpointId(param.Endpoint)
	if !exists {
		return nil
	}

	dsType, step, exists := store.LoadDsTypeAndStep(endpointId, param.Counter)
	if !exists {
		return nil
	}

	md5 := MUtils.Md5(param.Endpoint + "/" + param.Counter)
	filename := fmt.Sprintf("%s/%s/%s_%s_%d.rrd", g.Config().RRD.Storage, md5[0:2], md5, dsType, step)

	datas, err := rrdtool.Fetch(filename, param.ConsolFun, param.Start, param.End, step)
	if err != nil {
		items := store.GraphItems.PopAll(md5)
		size := len(items)
		if size > 2 {
			err := rrdtool.Flush(filename, items)
			if err != nil && g.Config().Debug && g.Config().DebugChecksum == md5 {
				log.Println("flush fail:", err, "filename:", filename)
			}
		} else {
			return nil
		}
	}
	items := store.GraphItems.FetchAll(md5)

	// merge
	items_size := len(items)
	datas_size := len(datas)
	if items_size > 1 && datas_size > 2 &&
		int(datas[1].Timestamp-datas[0].Timestamp) == step &&
		items[items_size-1].Timestamp > datas[0].Timestamp {

		var val model.JsonFloat
		cache_size := int(items[items_size-1].Timestamp-items[0].Timestamp)/step + 1
		cache := make([]*model.RRDData, cache_size, cache_size)

		//fix items
		items_idx := 0
		ts := items[0].Timestamp
		if dsType == g.DERIVE || dsType == g.COUNTER {
			for i := 0; i < cache_size; i++ {
				if items_idx < items_size-1 &&
					ts == items[items_idx].Timestamp &&
					ts != items[items_idx+1].Timestamp {
					val = model.JsonFloat(items[items_idx+1].Value-items[items_idx].Value) /
						model.JsonFloat(items[items_idx+1].Timestamp-items[items_idx].Timestamp)
					if val < 0 {
						val = model.JsonFloat(math.NaN())
					}
					items_idx++
				} else {
					// miss
					val = model.JsonFloat(math.NaN())
				}
				cache[i] = &model.RRDData{
					Timestamp: ts,
					Value:     val,
				}
				ts = ts + int64(step)
			}
		} else if dsType == g.GAUGE {
			for i := 0; i < cache_size; i++ {
				if items_idx < items_size && ts == items[items_idx].Timestamp {
					val = model.JsonFloat(items[items_idx].Value)
					items_idx++
				} else {
					// miss
					val = model.JsonFloat(math.NaN())
				}
				cache[i] = &model.RRDData{
					Timestamp: ts,
					Value:     val,
				}
				ts = ts + int64(step)
			}
		} else {
			log.Println("not support dstype")
			return nil
		}

		size := int(items[items_size-1].Timestamp-datas[0].Timestamp)/step + 1
		ret := make([]*model.RRDData, size, size)
		cache_idx := 0
		ts = datas[0].Timestamp

		if g.Config().Debug && g.Config().DebugChecksum == md5 {
			log.Println("param.start", param.Start, "param.End:", param.End,
				"items:", items, "datas:", datas)
		}

		for i := 0; i < size; i++ {
			if g.Config().Debug && g.Config().DebugChecksum == md5 {
				log.Println("i", i, "size:", size, "items_idx:", items_idx, "ts:", ts)
			}
			if i < datas_size {
				if ts == cache[cache_idx].Timestamp {
					if math.IsNaN(float64(cache[cache_idx].Value)) {
						val = datas[i].Value
					} else {
						val = cache[cache_idx].Value
					}
					cache_idx++
				} else {
					val = datas[i].Value
				}
			} else {
				if cache_idx < cache_size && ts == cache[cache_idx].Timestamp {
					val = cache[cache_idx].Value
					cache_idx++
				} else {
					//miss
					val = model.JsonFloat(math.NaN())
				}
			}
			ret[i] = &model.RRDData{
				Timestamp: ts,
				Value:     val,
			}
			ts = ts + int64(step)
		}
		resp.Values = ret
	} else {
		resp.Values = datas
	}

	resp.Endpoint = param.Endpoint
	resp.Counter = param.Counter
	resp.DsType = dsType
	resp.Step = step

	return nil
}

func (this *Graph) Info(param model.GraphInfoParam, resp *model.GraphInfoResp) error {
	endpointId, exists := store.LoadEndpointId(param.Endpoint)
	if !exists {
		return nil
	}

	dsType, step, exists := store.LoadDsTypeAndStep(endpointId, param.Counter)
	if !exists {
		return nil
	}

	md5 := MUtils.Md5(param.Endpoint + "/" + param.Counter)
	filename := fmt.Sprintf("%s/%s/%s_%s_%d.rrd", g.Config().RRD.Storage, md5[0:2], md5, dsType, step)

	resp.ConsolFun = dsType
	resp.Step = step
	resp.Filename = filename

	return nil
}
