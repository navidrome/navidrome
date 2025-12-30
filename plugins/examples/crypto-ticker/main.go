// Crypto Ticker Plugin - Demonstrates WebSocket host service capabilities.
//
// This plugin connects to Coinbase's WebSocket API to receive real-time
// cryptocurrency price updates and logs them to the Navidrome console.
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	pdk "github.com/extism/go-pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
	"github.com/navidrome/navidrome/plugins/pdk/go/websocket"
)

const (
	// Coinbase WebSocket API endpoint
	coinbaseWSEndpoint = "wss://ws-feed.exchange.coinbase.com"

	// Connection ID for our WebSocket connection
	connectionID = "crypto-ticker-conn"

	// ID for the reconnection schedule
	reconnectScheduleID = "crypto-ticker-reconnect"
)

// CoinbaseSubscription message structure
type CoinbaseSubscription struct {
	Type       string   `json:"type"`
	ProductIDs []string `json:"product_ids"`
	Channels   []string `json:"channels"`
}

// CoinbaseTicker message structure
type CoinbaseTicker struct {
	Type      string `json:"type"`
	Sequence  int64  `json:"sequence"`
	ProductID string `json:"product_id"`
	Price     string `json:"price"`
	Open24h   string `json:"open_24h"`
	Volume24h string `json:"volume_24h"`
	Low24h    string `json:"low_24h"`
	High24h   string `json:"high_24h"`
	BestBid   string `json:"best_bid"`
	BestAsk   string `json:"best_ask"`
	Time      string `json:"time"`
}

// cryptoTickerPlugin implements the lifecycle, websocket and scheduler interfaces.
type cryptoTickerPlugin struct{}

// init registers the plugin capabilities
func init() {
	lifecycle.Register(&cryptoTickerPlugin{})
	websocket.Register(&cryptoTickerPlugin{})
	scheduler.Register(&cryptoTickerPlugin{})
}

// Ensure cryptoTickerPlugin implements the required provider interfaces
var (
	_ lifecycle.InitProvider              = (*cryptoTickerPlugin)(nil)
	_ websocket.TextMessageProvider       = (*cryptoTickerPlugin)(nil)
	_ websocket.BinaryMessageProvider     = (*cryptoTickerPlugin)(nil)
	_ websocket.ErrorProvider             = (*cryptoTickerPlugin)(nil)
	_ websocket.CloseProvider             = (*cryptoTickerPlugin)(nil)
	_ scheduler.SchedulerCallbackProvider = (*cryptoTickerPlugin)(nil)
)

// OnInit is called when the plugin is loaded.
// We use this to establish the initial WebSocket connection.
func (p *cryptoTickerPlugin) OnInit(_ lifecycle.InitRequest) (lifecycle.InitResponse, error) {
	pdk.Log(pdk.LogInfo, "Crypto Ticker Plugin initializing...")

	// Get ticker configuration
	tickerConfig, ok := pdk.GetConfig("tickers")
	if !ok || tickerConfig == "" {
		tickerConfig = "BTC,ETH" // Default tickers
	}

	tickers := parseTickerSymbols(tickerConfig)
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Configured tickers: %v", tickers))

	// Connect to WebSocket
	err := connectAndSubscribe(tickers)
	if err != nil {
		pdk.Log(pdk.LogError, fmt.Sprintf("Failed to connect: %v", err))
		// Don't fail init - let reconnect logic handle it
	}

	return lifecycle.InitResponse{}, nil
}

// parseTickerSymbols parses a comma-separated list of ticker symbols
func parseTickerSymbols(tickerConfig string) []string {
	parts := strings.Split(tickerConfig, ",")
	tickers := make([]string, 0, len(parts))
	for _, ticker := range parts {
		ticker = strings.TrimSpace(ticker)
		if ticker == "" {
			continue
		}
		// Add -USD suffix if not present
		if !strings.Contains(ticker, "-") {
			ticker = ticker + "-USD"
		}
		tickers = append(tickers, ticker)
	}
	return tickers
}

// connectAndSubscribe connects to Coinbase WebSocket and subscribes to tickers
func connectAndSubscribe(tickers []string) error {
	// Connect to WebSocket using host function
	resp, err := host.WebSocketConnect(coinbaseWSEndpoint, nil, connectionID)
	if err != nil {
		return fmt.Errorf("WebSocket connection error: %w", err)
	}
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Connected to Coinbase WebSocket API (connection: %s)", resp.NewConnectionID))

	// Subscribe to ticker channel
	subscription := CoinbaseSubscription{
		Type:       "subscribe",
		ProductIDs: tickers,
		Channels:   []string{"ticker"},
	}

	subscriptionJSON, err := json.Marshal(subscription)
	if err != nil {
		return fmt.Errorf("JSON marshal error: %v", err)
	}

	// Send subscription message
	_, err = host.WebSocketSendText(connectionID, string(subscriptionJSON))
	if err != nil {
		return fmt.Errorf("WebSocket send error: %w", err)
	}

	pdk.Log(pdk.LogInfo, "Subscription message sent to Coinbase WebSocket API")
	return nil
}

// OnTextMessage is called when a text message is received
func (p *cryptoTickerPlugin) OnTextMessage(input websocket.OnTextMessageRequest) (websocket.OnTextMessageResponse, error) {
	// Only process messages from our connection
	if input.ConnectionID != connectionID {
		return websocket.OnTextMessageResponse{}, nil
	}

	// Try to parse as a ticker message
	var ticker CoinbaseTicker
	err := json.Unmarshal([]byte(input.Message), &ticker)
	if err != nil {
		// Not a valid JSON message, ignore
		return websocket.OnTextMessageResponse{}, nil
	}

	// Only process ticker messages
	if ticker.Type != "ticker" {
		// Could be subscription confirmation or heartbeat
		if ticker.Type != "" {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("Received %s message", ticker.Type))
		}
		return websocket.OnTextMessageResponse{}, nil
	}

	// Calculate 24h change percentage
	change := calculatePercentChange(ticker.Open24h, ticker.Price)

	// Log ticker information
	pdk.Log(pdk.LogInfo, fmt.Sprintf("💰 %s: $%s (24h: %s%%) Bid: $%s Ask: $%s",
		ticker.ProductID,
		ticker.Price,
		change,
		ticker.BestBid,
		ticker.BestAsk,
	))

	return websocket.OnTextMessageResponse{}, nil
}

// OnBinaryMessage is called when a binary message is received
func (p *cryptoTickerPlugin) OnBinaryMessage(input websocket.OnBinaryMessageRequest) (websocket.OnBinaryMessageResponse, error) {
	// Coinbase doesn't send binary messages, but we implement the handler anyway
	pdk.Log(pdk.LogWarn, fmt.Sprintf("Received unexpected binary message on connection %s", input.ConnectionID))
	return websocket.OnBinaryMessageResponse{}, nil
}

// OnError is called when an error occurs on the WebSocket connection
func (p *cryptoTickerPlugin) OnError(input websocket.OnErrorRequest) (websocket.OnErrorResponse, error) {
	pdk.Log(pdk.LogError, fmt.Sprintf("WebSocket error on connection %s: %s", input.ConnectionID, input.Error))
	return websocket.OnErrorResponse{}, nil
}

// OnClose is called when the WebSocket connection is closed
func (p *cryptoTickerPlugin) OnClose(input websocket.OnCloseRequest) (websocket.OnCloseResponse, error) {
	pdk.Log(pdk.LogInfo, fmt.Sprintf("WebSocket connection %s closed (code: %d, reason: %s)",
		input.ConnectionID, input.Code, input.Reason))

	// Only attempt reconnect for our connection
	if input.ConnectionID == connectionID {
		pdk.Log(pdk.LogInfo, "Scheduling reconnection attempt in 5 seconds...")

		// Schedule a one-time reconnection attempt
		_, err := host.SchedulerScheduleOneTime(5, "reconnect", reconnectScheduleID)
		if err != nil {
			pdk.Log(pdk.LogError, fmt.Sprintf("Failed to schedule reconnection: %v", err))
		}
	}

	return websocket.OnCloseResponse{}, nil
}

// OnSchedulerCallback is called when a scheduled task fires
func (p *cryptoTickerPlugin) OnSchedulerCallback(input scheduler.SchedulerCallbackRequest) (scheduler.SchedulerCallbackResponse, error) {
	// Only handle our reconnection schedule
	if input.ScheduleID != reconnectScheduleID {
		return scheduler.SchedulerCallbackResponse{}, nil
	}

	pdk.Log(pdk.LogInfo, "Attempting to reconnect to Coinbase WebSocket API...")

	// Get ticker configuration
	tickerConfig, ok := pdk.GetConfig("tickers")
	if !ok || tickerConfig == "" {
		tickerConfig = "BTC,ETH"
	}

	tickers := parseTickerSymbols(tickerConfig)

	// Try to connect and subscribe
	err := connectAndSubscribe(tickers)
	if err != nil {
		pdk.Log(pdk.LogError, fmt.Sprintf("Reconnection failed: %v - will retry in 10 seconds", err))

		// Schedule another attempt
		_, err := host.SchedulerScheduleOneTime(10, "reconnect", reconnectScheduleID)
		if err != nil {
			pdk.Log(pdk.LogError, fmt.Sprintf("Failed to schedule retry: %v", err))
		}
	} else {
		pdk.Log(pdk.LogInfo, "Successfully reconnected!")
	}

	return scheduler.SchedulerCallbackResponse{}, nil
}

// calculatePercentChange calculates the percentage change between open and current price
func calculatePercentChange(open, current string) string {
	var openFloat, currentFloat float64
	_, err := fmt.Sscanf(open, "%f", &openFloat)
	if err != nil || openFloat == 0 {
		return "N/A"
	}
	_, err = fmt.Sscanf(current, "%f", &currentFloat)
	if err != nil {
		return "N/A"
	}

	change := ((currentFloat - openFloat) / openFloat) * 100
	if change >= 0 {
		return fmt.Sprintf("+%.2f", change)
	}
	return fmt.Sprintf("%.2f", change)
}

func main() {}
