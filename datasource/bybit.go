package datasource

import (
	"log"
	"strconv"
	"time"

	"github.com/hirokisan/bybit/v2"
	"github.com/jiamingke/mock-trading-api/config"
)

func NewBybit(cfg config.BybitConfig, timeRangeCfg config.TimeRangeConfig) Datasource {
	return &bybitDatasource{
		client:          *bybit.NewClient().WithBaseURL(cfg.API.Url).WithAuth(cfg.API.Key, cfg.API.Secret),
		klineConfig:     cfg.Kline,
		startTimeMs:     timeRangeCfg.StartTimeMs,
		endTimeMs:       timeRangeCfg.EndTimeMs,
		nextStartTimeMs: timeRangeCfg.StartTimeMs,
	}
}

type bybitDatasource struct {
	client          bybit.Client
	klineConfig     config.BybitKline
	startTimeMs     int
	endTimeMs       int
	nextStartTimeMs int
	data            []KlineItem
	index           int
}

func (b *bybitDatasource) HasNext() bool {
	if !b.hasNext() {
		b.prepareData()
	}

	return b.hasNext()
}

func (b *bybitDatasource) Next() KlineItem {
	defer func() {
		b.index++
	}()

	return b.Current()
}

func (b bybitDatasource) Current() KlineItem {
	return b.data[b.index]
}

func (b bybitDatasource) hasNext() bool {
	return b.index < len(b.data)
}

func (b *bybitDatasource) prepareData() {
	b.index = 0

	if b.nextStartTimeMs > b.endTimeMs {
		b.data = nil
		return
	}

	klineItems, err := b.getKline(b.nextStartTimeMs, b.endTimeMs)
	if err != nil {
		log.Println(err)
		b.data = nil
		return
	}

	if len(klineItems) == 0 {
		b.data = nil
		return
	}

	lastStart, err := strconv.ParseInt(klineItems[len(klineItems)-1].StartTime, 10, 64)
	if err != nil {
		log.Println(err)
		b.data = nil
		return
	}

	b.data = klineItems
	b.nextStartTimeMs = int(lastStart) + int(time.Second/time.Millisecond)
}

func (b bybitDatasource) getKline(startTimeMs, endTimeMs int) ([]KlineItem, error) {
	resp, err := b.client.V5().Market().GetKline(bybit.V5GetKlineParam{
		Category: bybit.CategoryV5(b.klineConfig.Category),
		Symbol:   bybit.SymbolV5(b.klineConfig.Symbol),
		Interval: bybit.Interval(b.klineConfig.Interval),
		Start:    &startTimeMs,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	resultList := resp.Result.List
	length := len(resultList)

	klineItems := make([]KlineItem, length)
	for i, item := range resultList {
		klineItems[length-1-i] = KlineItem{
			StartTime: item.StartTime,
			Open:      item.Open,
			High:      item.High,
			Low:       item.Low,
			Close:     item.Close,
			Volume:    item.Volume,
			Turnover:  item.Turnover,
		}
	}

	// truncate the slice by binary search instead of checking the StartTime in the for loop
	// can improve the performance from O(N) to O(log N)
	index := length - 1
	endTime := strconv.Itoa(endTimeMs)
	for klineItems[index].StartTime > endTime {
		index /= 2
	}

	return klineItems[:index], nil
}
