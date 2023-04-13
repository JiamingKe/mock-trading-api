package handler

import "net/http"

type Handler interface {
	HandleWebSocketKline(w http.ResponseWriter, r *http.Request)
	HandlePlaceOrder(w http.ResponseWriter, r *http.Request)
}
