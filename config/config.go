package config

type Config struct {
	Bybit                *BybitConfig    `yaml:"bybit"`
	TimeRange            TimeRangeConfig `yaml:"time_range"`
	Fee                  float64         `yaml:"fee"`
	WebSocketKlinePath   string          `yaml:"ws_kline_path"`
	WebSocketPrivatePath string          `yaml:"ws_private_path"`
	CreateOrderPath      string          `yaml:"create_order_path"`
	SetTradingStopPath   string          `yaml:"set_trading_stop_path"`
	Port                 int             `yaml:"port"`
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
