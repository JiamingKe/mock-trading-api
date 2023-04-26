package datasource

import (
	"log"
	"strconv"
	"time"

	"github.com/hirokisan/bybit/v2"
	"github.com/jiamingke/mock-trading-api/config"
)

func NewBybit(cfg config.BybitConfig, timeRangeCfg config.TimeRangeConfig) Datasource {
	datasource := &bybitDatasource{
		client:          *bybit.NewClient().WithBaseURL(cfg.API.Url).WithAuth(cfg.API.Key, cfg.API.Secret),
		klineConfig:     cfg.Kline,
		timeRangeConfig: timeRangeCfg,
		klineItems:      make([]KlineItem, 0),
		klineIndex:      0,
	}

	datasource.loadKlineData()
	return datasource
}

type bybitDatasource struct {
	client          bybit.Client
	klineConfig     config.BybitKline
	timeRangeConfig config.TimeRangeConfig

	klineItems []KlineItem
	klineIndex int
}

func (b bybitDatasource) HasNext() bool {
	return b.klineIndex < len(b.klineItems)
}

func (b *bybitDatasource) Next() KlineItem {
	defer func() {
		b.klineIndex++
	}()

	return b.Get()
}

func (b *bybitDatasource) Get() KlineItem {
	return b.klineItems[b.klineIndex]
}

func (b *bybitDatasource) loadKlineData() {
	log.Println("loading kline data...")

	nextStartTimeMs := b.timeRangeConfig.StartTimeMs
	endTimeMs := b.timeRangeConfig.EndTimeMs

	for nextStartTimeMs < endTimeMs {
		klineItems, err := b.getKline(nextStartTimeMs, endTimeMs)
		if err != nil {
			log.Println(err)
			return
		}

		if len(klineItems) == 0 {
			break
		}

		lastStart, err := strconv.ParseInt(klineItems[len(klineItems)-1].StartTime, 10, 64)
		if err != nil {
			log.Println(err)
			return
		}

		b.klineItems = append(b.klineItems, klineItems...)
		nextStartTimeMs = int(lastStart) + +int(time.Second/time.Millisecond)
	}

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
