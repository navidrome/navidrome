//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/websocket"
)

const (
	// Coinbase WebSocket API endpoint
	coinbaseWSEndpoint = "wss://ws-feed.exchange.coinbase.com"

	// Connection ID for our WebSocket connection
	connectionID = "crypto-ticker-connection"
)

var (
	// Store ticker symbols from the configuration
	tickers []string
)

// WebSocketService instance used to manage WebSocket connections and communication.
var wsService = websocket.NewWebSocketService()

// CryptoTickerPlugin implements both WebSocketCallback and LifecycleManagement interfaces
type CryptoTickerPlugin struct{}

// Coinbase subscription message structure
type CoinbaseSubscription struct {
	Type       string   `json:"type"`
	ProductIDs []string `json:"product_ids"`
	Channels   []string `json:"channels"`
}

// Coinbase ticker message structure
type CoinbaseTicker struct {
	Type      string `json:"type"`
	Sequence  int64  `json:"sequence"`
	ProductID string `json:"product_id"`
	Price     string `json:"price"`
	Open24h   string `json:"open_24h"`
	Volume24h string `json:"volume_24h"`
	Low24h    string `json:"low_24h"`
	High24h   string `json:"high_24h"`
	Volume30d string `json:"volume_30d"`
	BestBid   string `json:"best_bid"`
	BestAsk   string `json:"best_ask"`
	Side      string `json:"side"`
	Time      string `json:"time"`
	TradeID   int    `json:"trade_id"`
	LastSize  string `json:"last_size"`
}

// OnInit is called when the plugin is loaded
func (CryptoTickerPlugin) OnInit(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	log.Printf("Crypto Ticker Plugin initializing...")

	// Check if ticker configuration exists
	tickerConfig, ok := req.Config["tickers"]
	if !ok {
		return &api.InitResponse{Error: "Missing 'tickers' configuration"}, nil
	}

	// Parse ticker symbols (comma-separated list)
	tickers = strings.Split(tickerConfig, ",")
	for i, ticker := range tickers {
		tickers[i] = strings.TrimSpace(ticker)

		// Add -USD suffix if not present
		if !strings.Contains(tickers[i], "-") {
			tickers[i] = tickers[i] + "-USD"
		}
	}

	log.Printf("Configured tickers: %v", tickers)

	// Connect to the WebSocket API using the WebSocketService interface
	_, err := wsService.Connect(ctx, &websocket.ConnectRequest{
		Url:          coinbaseWSEndpoint,
		ConnectionId: connectionID,
	})

	if err != nil {
		log.Printf("Failed to connect to Coinbase WebSocket API: %v", err)
		return &api.InitResponse{Error: fmt.Sprintf("WebSocket connection error: %v", err)}, nil
	}

	log.Printf("Connected to Coinbase WebSocket API")

	// Subscribe to ticker channel for the configured symbols
	subscription := CoinbaseSubscription{
		Type:       "subscribe",
		ProductIDs: tickers,
		Channels:   []string{"ticker"},
	}

	subscriptionJSON, err := json.Marshal(subscription)
	if err != nil {
		log.Printf("Failed to marshal subscription message: %v", err)
		return &api.InitResponse{Error: fmt.Sprintf("JSON marshal error: %v", err)}, nil
	}

	// Send subscription message
	_, err = wsService.SendText(ctx, &websocket.SendTextRequest{
		ConnectionId: connectionID,
		Message:      string(subscriptionJSON),
	})

	if err != nil {
		log.Printf("Failed to send subscription message: %v", err)
		return &api.InitResponse{Error: fmt.Sprintf("WebSocket send error: %v", err)}, nil
	}

	log.Printf("Subscription message sent to Coinbase WebSocket API")

	return &api.InitResponse{}, nil
}

// OnTextMessage is called when a text message is received from the WebSocket
func (CryptoTickerPlugin) OnTextMessage(ctx context.Context, req *api.OnTextMessageRequest) (*api.OnTextMessageResponse, error) {
	// Only process messages from our connection
	if req.ConnectionId != connectionID {
		log.Printf("Received message from unexpected connection: %s", req.ConnectionId)
		return &api.OnTextMessageResponse{}, nil
	}

	// Try to parse as a ticker message
	var ticker CoinbaseTicker
	err := json.Unmarshal([]byte(req.Message), &ticker)
	if err != nil {
		log.Printf("Failed to parse ticker message: %v", err)
		return &api.OnTextMessageResponse{}, nil
	}

	// If the message is not a ticker or has an error, just log it
	if ticker.Type != "ticker" {
		// This could be subscription confirmation or other messages
		log.Printf("Received non-ticker message: %s", req.Message)
		return &api.OnTextMessageResponse{}, nil
	}

	// Format and print ticker information
	log.Printf("CRYPTO TICKER: %s Price: %s Best Bid: %s Best Ask: %s 24h Change: %s%%\n",
		ticker.ProductID,
		ticker.Price,
		ticker.BestBid,
		ticker.BestAsk,
		calculatePercentChange(ticker.Open24h, ticker.Price),
	)

	return &api.OnTextMessageResponse{}, nil
}

// OnBinaryMessage is called when a binary message is received
func (CryptoTickerPlugin) OnBinaryMessage(ctx context.Context, req *api.OnBinaryMessageRequest) (*api.OnBinaryMessageResponse, error) {
	// Not expected from Coinbase WebSocket API
	return &api.OnBinaryMessageResponse{}, nil
}

// OnError is called when an error occurs on the WebSocket connection
func (CryptoTickerPlugin) OnError(ctx context.Context, req *api.OnErrorRequest) (*api.OnErrorResponse, error) {
	log.Printf("WebSocket error: %s", req.Error)
	return &api.OnErrorResponse{}, nil
}

// OnClose is called when the WebSocket connection is closed
func (CryptoTickerPlugin) OnClose(ctx context.Context, req *api.OnCloseRequest) (*api.OnCloseResponse, error) {
	log.Printf("WebSocket connection closed with code %d: %s", req.Code, req.Reason)

	// Try to reconnect if this is our connection
	if req.ConnectionId == connectionID {
		log.Printf("Attempting to reconnect to Coinbase WebSocket API...")

		// Connect to the WebSocket API
		_, err := wsService.Connect(ctx, &websocket.ConnectRequest{
			Url:          coinbaseWSEndpoint,
			ConnectionId: connectionID,
		})

		if err != nil {
			log.Printf("Failed to reconnect to Coinbase WebSocket API: %v", err)
			return &api.OnCloseResponse{}, nil
		}

		log.Printf("Reconnected to Coinbase WebSocket API")

		// Resubscribe to ticker channel
		subscription := CoinbaseSubscription{
			Type:       "subscribe",
			ProductIDs: tickers,
			Channels:   []string{"ticker"},
		}

		subscriptionJSON, err := json.Marshal(subscription)
		if err != nil {
			log.Printf("Failed to marshal subscription message: %v", err)
			return &api.OnCloseResponse{}, nil
		}

		// Send subscription message
		_, err = wsService.SendText(ctx, &websocket.SendTextRequest{
			ConnectionId: connectionID,
			Message:      string(subscriptionJSON),
		})

		if err != nil {
			log.Printf("Failed to send subscription message: %v", err)
			return &api.OnCloseResponse{}, nil
		}

		log.Printf("Resubscription message sent")
	}

	return &api.OnCloseResponse{}, nil
}

// Helper function to calculate percent change
func calculatePercentChange(open, current string) string {
	var openFloat, currentFloat float64
	_, err := fmt.Sscanf(open, "%f", &openFloat)
	if err != nil {
		return "N/A"
	}
	_, err = fmt.Sscanf(current, "%f", &currentFloat)
	if err != nil {
		return "N/A"
	}

	if openFloat == 0 {
		return "N/A"
	}

	change := ((currentFloat - openFloat) / openFloat) * 100
	return fmt.Sprintf("%.2f", change)
}

// Required by Go WASI build
func main() {}

func init() {
	api.RegisterWebSocketCallback(CryptoTickerPlugin{})
	api.RegisterLifecycleManagement(CryptoTickerPlugin{})
}
