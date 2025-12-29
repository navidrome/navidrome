// Crypto Ticker Plugin - Demonstrates WebSocket host service capabilities.
//
// This plugin connects to Coinbase's WebSocket API to receive real-time
// cryptocurrency price updates and logs them to the Navidrome console.
//
// Note: run `go doc -all` in this package to see all of the types and functions available.
// ./pdk.gen.go contains the domain types from the host where your plugin will run.
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	pdk "github.com/extism/go-pdk"
)

const (
	// Coinbase WebSocket API endpoint
	coinbaseWSEndpoint = "wss://ws-feed.exchange.coinbase.com"

	// Connection ID for our WebSocket connection
	connectionID = "crypto-ticker-conn"

	// ID for the reconnection schedule
	reconnectScheduleID = "crypto-ticker-reconnect"
)

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
	BestBid   string `json:"best_bid"`
	BestAsk   string `json:"best_ask"`
	Time      string `json:"time"`
}

// OnInitInput is the input for nd_on_init (currently empty, reserved for future use)
type OnInitInput struct{}

// OnInitOutput is the output from nd_on_init
type OnInitOutput struct {
	Error *string `json:"error,omitempty"`
}

// nd_on_init is called when the plugin is loaded.
// We use this to establish the initial WebSocket connection.
//
//export nd_on_init
func ndOnInit() int32 {
	// Read input (currently empty)
	var input OnInitInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return -1
	}

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

	// Return success output
	if err := pdk.OutputJSON(OnInitOutput{}); err != nil {
		pdk.SetError(err)
		return -1
	}
	return 0
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
	resp, err := WebSocketConnect(coinbaseWSEndpoint, nil, connectionID)
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
	_, err = WebSocketSendText(connectionID, string(subscriptionJSON))
	if err != nil {
		return fmt.Errorf("WebSocket send error: %w", err)
	}

	pdk.Log(pdk.LogInfo, "Subscription message sent to Coinbase WebSocket API")
	return nil
}

// NdWebsocketOnTextMessage is called when a text message is received
func NdWebsocketOnTextMessage(input OnTextMessageInput) (OnTextMessageOutput, error) {
	// Only process messages from our connection
	if input.ConnectionId != connectionID {
		return OnTextMessageOutput{}, nil
	}

	// Try to parse as a ticker message
	var ticker CoinbaseTicker
	err := json.Unmarshal([]byte(input.Message), &ticker)
	if err != nil {
		// Not a valid JSON message, ignore
		return OnTextMessageOutput{}, nil
	}

	// Only process ticker messages
	if ticker.Type != "ticker" {
		// Could be subscription confirmation or heartbeat
		if ticker.Type != "" {
			pdk.Log(pdk.LogDebug, fmt.Sprintf("Received %s message", ticker.Type))
		}
		return OnTextMessageOutput{}, nil
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

	return OnTextMessageOutput{}, nil
}

// NdWebsocketOnBinaryMessage is called when a binary message is received
func NdWebsocketOnBinaryMessage(input OnBinaryMessageInput) (OnBinaryMessageOutput, error) {
	// Coinbase doesn't send binary messages, but we implement the handler anyway
	pdk.Log(pdk.LogWarn, fmt.Sprintf("Received unexpected binary message on connection %s", input.ConnectionId))
	return OnBinaryMessageOutput{}, nil
}

// NdWebsocketOnError is called when an error occurs on the WebSocket connection
func NdWebsocketOnError(input OnErrorInput) (OnErrorOutput, error) {
	pdk.Log(pdk.LogError, fmt.Sprintf("WebSocket error on connection %s: %s", input.ConnectionId, input.Error))
	return OnErrorOutput{}, nil
}

// NdWebsocketOnClose is called when the WebSocket connection is closed
func NdWebsocketOnClose(input OnCloseInput) (OnCloseOutput, error) {
	pdk.Log(pdk.LogInfo, fmt.Sprintf("WebSocket connection %s closed (code: %d, reason: %s)",
		input.ConnectionId, input.Code, input.Reason))

	// Only attempt reconnect for our connection
	if input.ConnectionId == connectionID {
		pdk.Log(pdk.LogInfo, "Scheduling reconnection attempt in 5 seconds...")

		// Schedule a one-time reconnection attempt
		_, err := SchedulerScheduleOneTime(5, "reconnect", reconnectScheduleID)
		if err != nil {
			pdk.Log(pdk.LogError, fmt.Sprintf("Failed to schedule reconnection: %v", err))
		}
	}

	return OnCloseOutput{}, nil
}

// Scheduler callback input/output types
type SchedulerCallbackInput struct {
	ScheduleId  string `json:"scheduleId"`
	Payload     string `json:"payload"`
	IsRecurring bool   `json:"isRecurring"`
}

type SchedulerCallbackOutput struct {
	Error *string `json:"error,omitempty"`
}

// nd_scheduler_callback is called when a scheduled task fires
//
//export nd_scheduler_callback
func ndSchedulerCallback() int32 {
	var input SchedulerCallbackInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return -1
	}

	// Only handle our reconnection schedule
	if input.ScheduleId != reconnectScheduleID {
		return 0
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
		_, err := SchedulerScheduleOneTime(10, "reconnect", reconnectScheduleID)
		if err != nil {
			pdk.Log(pdk.LogError, fmt.Sprintf("Failed to schedule retry: %v", err))
		}
	} else {
		pdk.Log(pdk.LogInfo, "Successfully reconnected!")
	}

	output := SchedulerCallbackOutput{}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return -1
	}
	return 0
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
