# Crypto Ticker Plugin

This is a WebSocket-based WASM plugin for Navidrome that displays real-time cryptocurrency prices from Coinbase.

## Features

- Connects to Coinbase WebSocket API to receive real-time ticker updates
- Configurable to track multiple cryptocurrency pairs
- Implements WebSocket callback handlers for message processing
- Automatically reconnects on connection loss using the scheduler service
- Displays price, best bid, best ask, and 24-hour percentage change

## Configuration

Configure in the Navidrome UI (Settings → Plugins → crypto-ticker):

```json
{
  "tickers": "BTC,ETH,SOL,MATIC"
}
```

- `tickers` is a comma-separated list of cryptocurrency symbols
- The plugin will append `-USD` to any symbol without a trading pair specified
- Default: `BTC,ETH` if not configured

## How it Works

1. On plugin initialization, connects to Coinbase's WebSocket API
2. Subscribes to ticker updates for the configured cryptocurrencies
3. Incoming ticker data is processed via `nd_websocket_on_text_message` callback
4. On connection loss, schedules a reconnection attempt via the scheduler service
5. Reconnection is attempted until successful

## Building

This plugin was scaffolded using XTP CLI:

```bash
xtp plugin init --schema-file ../schemas/websocket_callback.yaml --template go --path ./crypto-ticker --name crypto-ticker
```

To build the plugin to WASM:

```bash
# Using TinyGo (recommended - smaller binary)
tinygo build -o crypto-ticker.wasm -target wasip1 -buildmode=c-shared .

# Or using standard Go
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o crypto-ticker.wasm .
```

## Installation

Copy the resulting `crypto-ticker.wasm` to your Navidrome plugins folder.

## Example Output

```
[Crypto] Crypto Ticker Plugin initializing...
[Crypto] Configured tickers: [BTC-USD ETH-USD]
[Crypto] Connected to Coinbase WebSocket API (connection: crypto-ticker-conn)
[Crypto] Subscription message sent to Coinbase WebSocket API
[Crypto] Received subscriptions message
[Crypto] 💰 BTC-USD: $98765.43 (24h: +2.35%) Bid: $98764.00 Ask: $98766.00
[Crypto] 💰 ETH-USD: $3456.78 (24h: -0.54%) Bid: $3455.90 Ask: $3457.80
```

## Permissions Required

- **config**: Read ticker symbols configuration
- **websocket**: Connect to `ws-feed.exchange.coinbase.com`
- **scheduler**: Schedule reconnection attempts

## Files

- `main.go` - Main plugin implementation
- `pdk.gen.go` - Generated WebSocket callback types (from XTP)
- `nd_host.go` - Host function wrappers for WebSocket and Scheduler services

---

For more details, see the source code in `main.go`.
