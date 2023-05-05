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
	"github.com/jiamingke/techan"
	"github.com/sdcoffey/big"
)

func NewBybit(bybitCfg config.BybitConfig, timeRangeCfg config.TimeRangeConfig, fee float64) Handler {
	return &bybitHandler{
		ds:         datasource.NewBybit(bybitCfg, timeRangeCfg),
		record:     techan.NewTradingRecord(),
		newRecord:  make(chan bool, 1),
		fee:        fee,
		takeProfit: big.ZERO,
		stopLoss:   big.ZERO,

		interval: bybitCfg.Kline.Interval,
		symbol:   bybitCfg.Kline.Symbol,
		orders:   make([]string, 0),
	}
}

type bybitHandler struct {
	ds         datasource.Datasource
	record     *techan.TradingRecord
	newRecord  chan bool
	fee        float64
	takeProfit big.Decimal
	stopLoss   big.Decimal

	interval string
	symbol   string
	orders   []string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *bybitHandler) HandleWebSocketKline(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection to WebSocket:", err)
		return
	}
	defer conn.Close()

	conn.SetPingHandler(func(appData string) error {
		err := conn.WriteMessage(websocket.PongMessage, nil)
		if err != nil {
			log.Println("Failed to send pong message:", err)
		}

		return nil
	})

	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read incoming message:", err)
				conn.Close()
				return
			}
		}
	}()

	ticker := time.NewTicker(25 * time.Millisecond)

	for {
		<-ticker.C

		if !h.ds.HasNext() {
			for _, s := range h.orders {
				fmt.Println(s)
			}

			os.Exit(0)
		}

		h.fillPositionWithSLTP()

		// some buffer time for the updated position to reach the client side
		time.Sleep(3 * time.Millisecond)

		h.websocketWriteKline(conn)
	}
}

func (h *bybitHandler) fillPositionWithSLTP() {
	if !h.ds.HasNext() || h.record.CurrentPosition().IsNew() {
		return
	}

	klineItem := h.ds.Get()
	position := h.record.CurrentPosition()

	if !h.stopLoss.EQ(big.ZERO) && !h.takeProfit.EQ(big.ZERO) {
		high := big.NewFromString(klineItem.High)
		low := big.NewFromString(klineItem.Low)

		if position.IsLong() {
			if high.GTE(h.takeProfit) {
				h.createOrder(techan.SELL, h.takeProfit, position.EntranceOrder().Amount, big.ZERO, big.ZERO)
			} else if low.LTE(h.stopLoss) {
				h.createOrder(techan.SELL, h.stopLoss, position.EntranceOrder().Amount, big.ZERO, big.ZERO)
			}
		} else if position.IsShort() {
			if low.LTE(h.takeProfit) {
				h.createOrder(techan.BUY, h.takeProfit, position.EntranceOrder().Amount, big.ZERO, big.ZERO)
			} else if high.GTE(h.stopLoss) {
				h.createOrder(techan.BUY, h.stopLoss, position.EntranceOrder().Amount, big.ZERO, big.ZERO)
			}
		}
	}

}

func (h *bybitHandler) websocketWriteKline(conn *websocket.Conn) {
	if !h.ds.HasNext() {
		return
	}

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
	}

	err = conn.WriteMessage(websocket.TextMessage, bytes)
	if err != nil {
		log.Println("failed to write message to WebSocket:", err)
		os.Exit(1)
	}

}

func (h *bybitHandler) HandleWebSocketPrivate(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection to WebSocket:", err)
		return
	}
	defer conn.Close()

	// auth
	_, _, err = conn.ReadMessage()
	if err != nil {
		log.Println("Failed to authorise the connection:", err)
		return
	}

	authResp, err := json.Marshal(map[string]interface{}{
		"success": true,
	})
	if err != nil {
		log.Println("Failed to parse the authorisation response:", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, authResp)
	if err != nil {
		log.Println("Failed to write the authorisation response:", err)
		return
	}

	go func() {
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read incoming message:", err)
				conn.Close()
				os.Exit(1)
				return
			}

			if string(p) == `{"op":"ping"}` {
				err := conn.WriteMessage(websocket.PongMessage, nil)
				if err != nil {
					log.Println("Failed to send pong message:", err)
					conn.Close()
					os.Exit(1)
					return
				}
			}
		}
	}()

	for {
		<-h.newRecord
		h.websocketWritePosition(conn)
	}
}

func (h *bybitHandler) websocketWritePosition(conn *websocket.Conn) {

	data := bybit.V5WebsocketPrivatePositionData{
		Symbol:      bybit.SymbolV5BTCUSDT,
		Category:    bybit.CategoryV5Linear,
		UpdatedTime: fmt.Sprintf("%d", time.Now().UTC().UnixMilli()),
	}

	switch {
	case h.record.CurrentPosition().IsNew():
		data.Side = bybit.SideNone
	case h.record.CurrentPosition().IsClosed():
		log.Println("position closed !")
		data.Side = bybit.SideNone
	case h.record.CurrentPosition().IsOpen():
		if h.record.CurrentPosition().IsLong() {
			data.Side = bybit.SideBuy
		} else {
			data.Side = bybit.SideSell
		}

		if !h.takeProfit.EQ(big.ZERO) {
			data.TakeProfit = h.takeProfit.FormattedString(4)
		}

		if !h.stopLoss.EQ(big.ZERO) {
			data.StopLoss = h.stopLoss.FormattedString(4)
		}

		data.EntryPrice = h.record.CurrentPosition().EntranceOrder().Price.FormattedString(2)
		data.Size = h.record.CurrentPosition().EntranceOrder().Amount.FormattedString(4)
	default:
		log.Println("unexpected write position case")
	}

	response := bybit.V5WebsocketPrivatePositionResponse{
		Topic: bybit.V5WebsocketPrivateTopicPosition,
		Data:  []bybit.V5WebsocketPrivatePositionData{data},
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		log.Println("failed to marshal the response:", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, bytes)
	if err != nil {
		log.Println("failed to write message to WebSocket:", err)
		os.Exit(1)
	}

}

func (h *bybitHandler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	// Bind the request body to a struct
	var req bybit.V5CreateOrderParam
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var side techan.OrderSide
	if req.Side == bybit.SideSell {
		side = techan.SELL
	}

	entryPrice := big.NewFromString(h.ds.Get().Open)

	resp := h.createOrder(side, entryPrice, big.NewFromString(req.Qty), big.NewFromString(*req.TakeProfit), big.NewFromString(*req.StopLoss))

	// Return a success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *bybitHandler) createOrder(side techan.OrderSide, entryPrice, qty, takeProfit, stopLoss big.Decimal) bybit.V5CreateOrderResponse {
	kline := h.ds.Get()

	h.takeProfit = takeProfit
	h.stopLoss = stopLoss

	cost := entryPrice.Mul(qty).Mul(big.NewDecimal(h.fee))
	t, _ := strconv.ParseInt(kline.StartTime, 10, 64)
	executionTime := time.UnixMilli(t)

	// write to orders.json
	h.orders = append(h.orders, fmt.Sprintf("{\"side\": \"%+v\", \"qty\":\"%s\", \"cost\":\"%s\", \"timestamp\": \"%s\", \"price\":\"%s\", \"takeProfit\": \"%s\", \"stopLoss\": \"%s\"}", side, qty, cost, executionTime, entryPrice, takeProfit, stopLoss))

	order := techan.Order{
		Security:      "Linear.BTC/USDT",
		Side:          side,
		Price:         entryPrice,
		Amount:        qty,
		ExecutionTime: executionTime,
	}

	h.record.Operate(order)

	if len(h.newRecord) == 0 {
		h.newRecord <- true
	}

	return bybit.V5CreateOrderResponse{}
}

func (h *bybitHandler) HandleSetTradingStop(w http.ResponseWriter, r *http.Request) {
	// Return a success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bybit.V5SetTradingStopResponse{})
}
