package config

type Config struct {
	Bybit                   *BybitConfig    `yaml:"bybit"`
	TimeRange               TimeRangeConfig `yaml:"time_range"`
	WebSocketKlinePath      string          `yaml:"ws_kline_path"`
	WebSocketKlineLatencyMs int             `yaml:"ws_kline_latency_ms"`
	PlaceOrderPath          string          `yaml:"place_order_path"`
	Port                    int             `yaml:"port"`
}

type BybitConfig struct {
	API   BybitAPI   `yaml:"api"`
	Kline BybitKline `yaml:"kline"`
}

type TimeRangeConfig struct {
	StartTimeMs int `yaml:"start_time_ms"`
	EndTimeMs   int `yaml:"end_time_ms"`
}

type BybitAPI struct {
	Url    string `yaml:"url"`
	Key    string `yaml:"key"`
	Secret string `yaml:"secret"`
}

type BybitKline struct {
	Category string `yaml:"category"`
	Symbol   string `yaml:"symbol"`
	Interval string `yaml:"interval"`
}
