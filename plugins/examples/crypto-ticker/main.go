// Crypto Ticker Plugin - Demonstrates WebSocket host service capabilities.
//
// This plugin connects to Coinbase's WebSocket API to receive real-time
// cryptocurrency price updates and logs them to the Navidrome console.
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
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

	// Config keys (must match manifest.json schema property names)
	symbolsKey        = "symbols"
	reconnectDelayKey = "reconnectDelay"
	logPricesKey      = "logPrices"

	// Default values
	defaultReconnectDelay = 5
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
	_ lifecycle.InitProvider          = (*cryptoTickerPlugin)(nil)
	_ websocket.TextMessageProvider   = (*cryptoTickerPlugin)(nil)
	_ websocket.BinaryMessageProvider = (*cryptoTickerPlugin)(nil)
	_ websocket.ErrorProvider         = (*cryptoTickerPlugin)(nil)
	_ websocket.CloseProvider         = (*cryptoTickerPlugin)(nil)
	_ scheduler.CallbackProvider      = (*cryptoTickerPlugin)(nil)
)

// OnInit is called when the plugin is loaded.
// We use this to establish the initial WebSocket connection.
func (p *cryptoTickerPlugin) OnInit() error {
	pdk.Log(pdk.LogInfo, "Crypto Ticker Plugin initializing...")

	// Get ticker configuration from JSON schema config
	symbols := getSymbols()
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Configured symbols: %v", symbols))

	// Connect to WebSocket
	// Errors won't fail init - reconnect logic will handle it
	return connectAndSubscribe(symbols)
}

// getSymbols reads the symbols array from config
func getSymbols() []string {
	defaultSymbols := []string{"BTC-USD"}
	symbolsJSON, ok := pdk.GetConfig(symbolsKey)
	if !ok || symbolsJSON == "" {
		return defaultSymbols
	}

	var symbols []string
	if err := json.Unmarshal([]byte(symbolsJSON), &symbols); err != nil {
		pdk.Log(pdk.LogWarn, fmt.Sprintf("failed to parse symbols config: %v, using defaults", err))
		return defaultSymbols
	}

	if len(symbols) == 0 {
		return defaultSymbols
	}

	// Normalize symbols - add -USD suffix if not present
	for i, s := range symbols {
		s = strings.TrimSpace(s)
		if !strings.Contains(s, "-") {
			symbols[i] = s + "-USD"
		} else {
			symbols[i] = s
		}
	}

	return symbols
}

// getReconnectDelay reads the reconnect delay from config
func getReconnectDelay() int32 {
	delayStr, ok := pdk.GetConfig(reconnectDelayKey)
	if !ok || delayStr == "" {
		return defaultReconnectDelay
	}

	var delay int
	if _, err := fmt.Sscanf(delayStr, "%d", &delay); err != nil || delay < 1 {
		return defaultReconnectDelay
	}
	return int32(delay)
}

// shouldLogPrices reads the logPrices setting from config
func shouldLogPrices() bool {
	logStr, ok := pdk.GetConfig(logPricesKey)
	if !ok || logStr == "" {
		return false
	}
	return logStr == "true"
}

// connectAndSubscribe connects to Coinbase WebSocket and subscribes to tickers
func connectAndSubscribe(tickers []string) error {
	// Connect to WebSocket using host function
	newConnID, err := host.WebSocketConnect(coinbaseWSEndpoint, nil, connectionID)
	if err != nil {
		return fmt.Errorf("WebSocket connection error: %w", err)
	}
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Connected to Coinbase WebSocket API (connection: %s)", newConnID))

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
	err = host.WebSocketSendText(connectionID, string(subscriptionJSON))
	if err != nil {
		return fmt.Errorf("WebSocket send error: %w", err)
	}

	pdk.Log(pdk.LogInfo, "Subscription message sent to Coinbase WebSocket API")
	return nil
}

// OnTextMessage is called when a text message is received
func (p *cryptoTickerPlugin) OnTextMessage(input websocket.OnTextMessageRequest) error {
	// Only process messages from our connection
	if input.ConnectionID != connectionID {
		return nil
	}

	// Try to parse as a ticker message
	var ticker CoinbaseTicker
	err := json.Unmarshal([]byte(input.Message), &ticker)
	if err != nil {
		// Not a valid JSON message, ignore
		return nil
	}

	// Only process ticker messages
	if ticker.Type != "ticker" {
		// Could be subscription confirmation or heartbeat
		if ticker.Type != "" {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("Received %s message", ticker.Type))
		}
		return nil
	}

	// Calculate 24h change percentage
	change := calculatePercentChange(ticker.Open24h, ticker.Price)

	// Log ticker information (only if enabled in config)
	if shouldLogPrices() {
		pdk.Log(pdk.LogInfo, fmt.Sprintf("💰 %s: $%s (24h: %s%%) Bid: $%s Ask: $%s",
			ticker.ProductID,
			ticker.Price,
			change,
			ticker.BestBid,
			ticker.BestAsk,
		))
	}

	return nil
}

// OnBinaryMessage is called when a binary message is received
func (p *cryptoTickerPlugin) OnBinaryMessage(input websocket.OnBinaryMessageRequest) error {
	// Coinbase doesn't send binary messages, but we implement the handler anyway
	pdk.Log(pdk.LogWarn, fmt.Sprintf("Received unexpected binary message on connection %s", input.ConnectionID))
	return nil
}

// OnError is called when an error occurs on the WebSocket connection
func (p *cryptoTickerPlugin) OnError(input websocket.OnErrorRequest) error {
	pdk.Log(pdk.LogError, fmt.Sprintf("WebSocket error on connection %s: %s", input.ConnectionID, input.Error))
	return nil
}

// OnClose is called when the WebSocket connection is closed
func (p *cryptoTickerPlugin) OnClose(input websocket.OnCloseRequest) error {
	pdk.Log(pdk.LogInfo, fmt.Sprintf("WebSocket connection %s closed (code: %d, reason: %s)",
		input.ConnectionID, input.Code, input.Reason))

	// Only attempt reconnect for our connection
	if input.ConnectionID == connectionID {
		delay := getReconnectDelay()
		pdk.Log(pdk.LogInfo, fmt.Sprintf("Scheduling reconnection attempt in %d seconds...", delay))

		// Schedule a one-time reconnection attempt
		_, err := host.SchedulerScheduleOneTime(delay, "reconnect", reconnectScheduleID)
		if err != nil {
			pdk.Log(pdk.LogError, fmt.Sprintf("Failed to schedule reconnection: %v", err))
		}
	}

	return nil
}

// OnCallback is called when a scheduled task fires
func (p *cryptoTickerPlugin) OnCallback(input scheduler.SchedulerCallbackRequest) error {
	// Only handle our reconnection schedule
	if input.ScheduleID != reconnectScheduleID {
		return nil
	}

	pdk.Log(pdk.LogInfo, "Attempting to reconnect to Coinbase WebSocket API...")

	// Get ticker configuration
	symbols := getSymbols()

	// Try to connect and subscribe
	err := connectAndSubscribe(symbols)
	if err != nil {
		delay := getReconnectDelay() * 2 // Double delay on failure
		pdk.Log(pdk.LogError, fmt.Sprintf("Reconnection failed: %v - will retry in %d seconds", err, delay))

		// Schedule another attempt
		_, err := host.SchedulerScheduleOneTime(delay, "reconnect", reconnectScheduleID)
		if err != nil {
			pdk.Log(pdk.LogError, fmt.Sprintf("Failed to schedule retry: %v", err))
		}
	} else {
		pdk.Log(pdk.LogInfo, "Successfully reconnected!")
	}

	return nil
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
