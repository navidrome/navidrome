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

| Key       | Description                                                          | Default   |
|-----------|----------------------------------------------------------------------|-----------|
| `tickers` | Comma-separated list of cryptocurrency symbols (e.g., `BTC,ETH,SOL`) | `BTC,ETH` |

The plugin will append `-USD` to any symbol without a trading pair specified.

## How it Works

1. On plugin initialization, connects to Coinbase's WebSocket API
2. Subscribes to ticker updates for the configured cryptocurrencies
3. Incoming ticker data is processed via `nd_websocket_on_text_message` callback
4. On connection loss, schedules a reconnection attempt via the scheduler service
5. Reconnection is attempted until successful

## Building

To build the plugin and package as `.ndp`:

```bash
# Using TinyGo (recommended - smaller binary)
tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared .
zip -j crypto-ticker.ndp manifest.json plugin.wasm
```

Or from the `plugins/examples/` directory:

```bash
make crypto-ticker.ndp
```

## Installation

Copy the resulting `crypto-ticker.ndp` to your Navidrome plugins folder.

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
- `go.mod` - Go module file

## PDK

This plugin imports the Navidrome PDK subpackages directly:

```go
import (
    "github.com/navidrome/navidrome/plugins/pdk/go/host"
    "github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
    "github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
    "github.com/navidrome/navidrome/plugins/pdk/go/websocket"
)
```

The `go.mod` file uses `replace` directives to point to the local packages for development.

---

For more details, see the source code in `main.go`.
