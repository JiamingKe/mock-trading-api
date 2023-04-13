package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hirokisan/bybit/v2"
	"github.com/jiamingke/mock-trading-api/config"
	"github.com/jiamingke/mock-trading-api/datasource"
	"github.com/sdcoffey/big"
)

func NewBybit(bybitCfg config.BybitConfig, timeRangeCfg config.TimeRangeConfig, wsKlineLatencyMs int) Handler {
	return bybitHandler{
		ds:        datasource.NewBybit(bybitCfg, timeRangeCfg),
		interval:  bybitCfg.Kline.Interval,
		symbol:    bybitCfg.Kline.Symbol,
		latencyMs: wsKlineLatencyMs,
	}
}

type bybitHandler struct {
	ds        datasource.Datasource
	interval  string
	symbol    string
	latencyMs int
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h bybitHandler) HandleWebSocketKline(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection to WebSocket:", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(time.Duration(int(h.latencyMs)) * time.Millisecond)

	for h.ds.HasNext() {
		<-ticker.C

		klineItem := h.ds.Next()

		start, _ := strconv.ParseInt(klineItem.StartTime, 10, 64)
		response := bybit.V5WebsocketPublicKlineResponse{
			Topic:     fmt.Sprintf("kline.%s.%s", h.interval, h.symbol),
			Type:      "snapshot",
			TimeStamp: start,
			Data: []bybit.V5WebsocketPublicKlineData{
				{
					Start:     int(start),
					Open:      klineItem.Open,
					Close:     klineItem.Close,
					High:      klineItem.High,
					Low:       klineItem.Low,
					Volume:    klineItem.Volume,
					Confirm:   true,
					Timestamp: int(start),
				},
			},
		}
		bytes, err := json.Marshal(response)
		if err != nil {
			log.Println("failed to marshal the response:", err)
			break
		}

		err = conn.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			log.Println("failed to write message to WebSocket:", err)
			break
		}
	}

	os.Exit(0)
}

func (h bybitHandler) HandlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	// Bind the request body to a struct
	var req bybit.V5CreateOrderParam
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// entryPrice := big.NewFromString(h.ds.Current().High).Add(big.NewFromString(h.ds.Current().Low)).Div(big.NewFromInt(2))
	entryPrice := big.NewFromString(h.ds.Current().Open)
	qty := big.NewFromString(req.Qty)
	cost := entryPrice.Mul(qty).Mul(big.NewDecimal(0.0006))

	fmt.Printf("{\"side\": \"%s\", \"qty\":\"%s\", \"price\":\"%s\", \"cost\":\"%s\", \"timestamp\": \"%s\"}\n", req.Side, qty, entryPrice, cost, h.ds.Current().StartTime)

	if err != nil {
		// Return an error response if there's an error fetching data from the data source
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Return a success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bybit.V5CreateOrderResponse{})
}
