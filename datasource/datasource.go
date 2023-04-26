package datasource

type Datasource interface {
	HasNext() bool
	Next() KlineItem
	Get() KlineItem
}

type KlineItem struct {
	StartTime string
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
	Turnover  string
}
