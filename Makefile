build:
	@go build -o mock-trading-api ./

run:
	@./mock-trading-api cfg/bybit.yaml > orders.json

pnl:
	@python3 pnl.py

all: build run pnl
