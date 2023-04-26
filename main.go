package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jiamingke/mock-trading-api/config"
	"github.com/jiamingke/mock-trading-api/handler"
	"gopkg.in/yaml.v3"
)

func main() {

	args := os.Args

	if len(args) == 1 {
		log.Fatal("missing a config file to run the program")
		return
	}

	data, err := os.ReadFile(args[1])
	if err != nil {
		log.Fatal(err)
		return
	}

	var cfg config.Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatal(err)
		return
	}

	if cfg.Bybit == nil {
		log.Fatal("currently only support bybit API but the config is missing")
		return
	}

	handler := handler.NewBybit(*cfg.Bybit, cfg.TimeRange, cfg.Fee, cfg.WebSocketKlineLatencyMs)

	// Serve requests
	r := mux.NewRouter()

	r.HandleFunc(cfg.WebSocketKlinePath, handler.HandleWebSocketKline)
	r.HandleFunc(cfg.WebSocketPrivatePath, handler.HandleWebSocketPrivate)
	r.HandleFunc(cfg.CreateOrderPath, handler.HandleCreateOrder).Methods(http.MethodPost)
	r.HandleFunc(cfg.SetTradingStopPath, handler.HandleSetTradingStop).Methods(http.MethodPost)

	// Start HTTP server
	address := fmt.Sprintf(":%d", cfg.Port)
	log.Println("starting the server that listens to the address", address)

	httpServer := &http.Server{
		Addr:    address,
		Handler: r,
	}

	log.Fatal(httpServer.ListenAndServe())
}
