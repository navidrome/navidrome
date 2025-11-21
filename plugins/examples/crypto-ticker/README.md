# Crypto Ticker Plugin

This is a WebSocket-based WASM plugin for Navidrome that displays real-time cryptocurrency prices from Coinbase.

## Features

- Connects to Coinbase WebSocket API to receive real-time ticker updates
- Configurable to track multiple cryptocurrency pairs
- Implements WebSocketCallback and LifecycleManagement interfaces
- Automatically reconnects on connection loss
- Displays price, best bid, best ask, and 24-hour percentage change

## Configuration

In your `navidrome.toml` file, add:

```toml
[PluginConfig.crypto-ticker]
tickers = "BTC,ETH,SOL,MATIC"
```

- `tickers` is a comma-separated list of cryptocurrency symbols
- The plugin will append `-USD` to any symbol without a trading pair specified

## How it Works

- The plugin connects to Coinbase's WebSocket API upon initialization
- It subscribes to ticker updates for the configured cryptocurrencies
- Incoming ticker data is processed and logged
- On connection loss, it automatically attempts to reconnect (TODO)

## Building

To build the plugin to WASM:

```
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm plugin.go
```

## Installation

Copy the resulting `plugin.wasm` and create a `manifest.json` file in your Navidrome plugins folder under a `crypto-ticker` directory.

## Example Output

```
CRYPTO TICKER: BTC-USD Price: 65432.50 Best Bid: 65431.25 Best Ask: 65433.75 24h Change: 2.75%
CRYPTO TICKER: ETH-USD Price: 3456.78 Best Bid: 3455.90 Best Ask: 3457.80 24h Change: 1.25%
```

---

For more details, see the source code in `plugin.go`.
