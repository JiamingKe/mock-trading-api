package handler

import "net/http"

type Handler interface {
	HandleWebSocketKline(w http.ResponseWriter, r *http.Request)
	HandleWebSocketPrivate(w http.ResponseWriter, r *http.Request)

	HandleCreateOrder(w http.ResponseWriter, r *http.Request)
	HandleSetTradingStop(w http.ResponseWriter, r *http.Request)
}
