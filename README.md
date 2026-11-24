# Miroir Trading Service

> This repository is a read-only mirror of an internal repository, published at each release. Issues and pull requests are not accepted. To report a security issue, see [SECURITY.md](SECURITY.md).

## Overview

Miroir Trading Service copies the positions of Hyperliquid traders to your own Binance, Bybit or Hyperliquid accounts. Which traders to copy and how (exposure, leverage, excluded symbols) is configured in [Miroir](https://getmiroir.xyz) and fetched from the Control API. The service runs in the background as an OS service, and a desktop app manages login, exchange accounts and the service.

## Features

Each feature is marked with where it runs: **Local** (code in this repository, on your machine) or **Miroir API** (relies on the Miroir Control API, which is not part of this repository). See [What depends on Miroir](#what-depends-on-miroir) for the details.

### Supported exchanges

|                                 | Hyperliquid          | Binance | Bybit                  |
| ------------------------------- | :------------------: | :-----: | :--------------------: |
| Copy source (traders)           | ✅                   |         |                        |
| Copy destination (your account) | ✅ incl. subaccounts | ✅      | ✅ with regions        |
| Demo accounts                   |                      | ✅      | ✅                     |
| Testnet                         | ✅                   |         | ✅ develop builds only |

### Copy trading

| Feature | Description | Runs on |
| --- | --- | :---: |
| **Copy targets** | Traders, exposure factor, minimum equity, leverage overrides, excluded symbols and TP/SL safety, managed in Miroir | Miroir API |
| **Position sizing** | The ratio between your account and the trader's is computed by the Miroir API; positions are then sized locally from it and the exposure factor | Miroir API + Local |
| **Symbol mapping** | Hyperliquid-to-exchange symbol table fetched from the Miroir API, with local overrides and excluded symbols | Miroir API + Local |
| **Order execution** | Large orders split into fractions, minimum notional handling, optional websocket order routing | Local |
| **Concurrency** | Accounts and symbols synchronized concurrently | Local |

### Risk management

| Feature | Description | Runs on |
| --- | --- | :---: |
| **Stop loss / take profit** | Placed automatically and readjusted as positions and prices move. Stop losses are set so that, if all of them trigger, the loss stays within `max_fraction_at_risk` of the equity above the minimum equity floor. Position sizes are not affected. | Local |
| **Minimum equity** | The higher of the account's and the target's minimum equity. Below it, copied positions are closed (see Automatic close-out). | Local + Miroir API |
| **Leverage limits** | Maximum leverage per account and per symbol, with per-target overrides | Local |
| **Automatic close-out** | Closes all copied positions when the account falls below its minimum equity or account data can't be read, or when Miroir requests it (emergency close, outdated app version blocked) | Local + Miroir API |
| **Account validation** | Credentials and API key permissions checked when an account is added in the app, and at startup of the headless service | Local |

### Reliability and security

| Feature | Description | Runs on |
| --- | --- | :---: |
| **Background service** | Runs as a systemd, launchd or Windows service, with a health heartbeat | Local |
| **Rate limiting and retries** | Per-exchange rate limits, retries with exponential backoff and jitter | Local |
| **Secure credentials** | Secrets kept in the OS keyring (DPAPI-encrypted store on Windows) and redacted from log files | Local |
| **Logging** | Rotating log files | Local |
| **Metrics** | Sync metrics batched and sent to Miroir | Miroir API |

### Desktop app

| Feature | Description | Runs on |
| --- | --- | :---: |
| **Login** | Browser login on Miroir, returning to the app through `miroir://` deep links | Miroir API |
| **Exchange accounts** | Add and remove Binance, Bybit and Hyperliquid accounts locally; account labels (rename) are stored in Miroir | Local + Miroir API |
| **Service control** | Install, start, stop and monitor the background service | Local |
| **Updates** | Notifies when a new version is available | Miroir website |
| **Problem reports** | Send a report from the app, with logs optionally attached | Miroir API |
| **System tray** | Keeps running in the tray when the window is closed | Local |

## What depends on Miroir

This repository contains the full client: the copy loop, order execution, risk management, the exchange clients and the desktop app. Some parts rely on the Miroir Control API (`https://getmiroir.xyz/api`, configurable with `control_api.url`), which is not open source. A Miroir account is required to run the service.

| Feature | Endpoint | Sent to Miroir | Received |
| --- | --- | --- | --- |
| **Login** | `/auth/exchange`, `/auth/verify`, `/api-keys/*` | One-time login code; creation and revocation of this device's API key | A Miroir API key, stored in the OS keyring |
| **Copy targets** | `/control/targets` | Nothing beyond authentication | Traders to copy, their settings and each copy's status (including emergency close and blocked app versions) |
| **Position sizing** | `/control/compute-normalization` | Your account's current value; the trader's public Hyperliquid data | Normalization ratio and target value |
| **Symbol mapping** | `/symbol-mappings/` | Nothing beyond authentication | Hyperliquid-to-exchange symbol table |
| **Account labels** | `/accounts/labels`, `/accounts/set-label` | The label you set for an account | Your account labels |
| **Metrics** | `/control/push-loop-metrics` | For each sync: app version and OS; for each account its name, exchange, exchange user ID, API key expiry, equity, balance, PnL and margin; positions (size, prices, TP/SL, leverage); errors | Nothing |
| **Problem reports** | `/control/copy-service-report` | Your message, the user agent and, if you choose, the service and app logs (secrets redacted) | Nothing |
| **Updates** | `https://getmiroir.xyz/manifest.json` | Nothing | Latest version information |

Your exchange API keys and secrets and your Hyperliquid private key never leave your machine: they are only used to sign requests sent directly to the exchanges. Trader positions are read directly from the public Hyperliquid API.

## Don't trust our builds? Build your own

The service handles your exchange API keys, so you shouldn't have to take our word for what it does with them. Every release we publish is in this repository: read the code, build the app on your own computer and run that instead of our download. It works with your Miroir account exactly like the official app.

1. Install the [build requirements](#building), then check out the release you want to run:

   ```bash
   git clone https://github.com/getmiroir/trading-service.git
   cd trading-service
   git checkout v2.0   # replace with the release you want; `git tag` lists them
   ```

2. Build the desktop app, stamping it with that release version:

   ```bash
   cd gui
   wails build -ldflags "-X github.com/getmiroir/trading-service/internal/version.Version=v2.0 -X github.com/getmiroir/trading-service/internal/version.Platform=$(go env GOOS)/$(go env GOARCH)"
   ```

   On Linux, add `-tags webkit2_41` if needed (see [Build on Linux](#build-on-linux)). **Don't skip the version flag: without it the build is a `develop` build, which the Miroir API rejects. The service then stops copying and closes the copied positions.**

3. If the official app is installed, remove its background service first. The service runs the executable it was installed from, so otherwise it keeps running our build. With the official app closed, run your build with `-service uninstall`:

   ```bash
   gui/build/bin/miroir-gui -service uninstall                              # Linux
   gui/build/bin/miroir-gui.app/Contents/MacOS/miroir-gui -service uninstall # macOS
   gui\build\bin\miroir-gui.exe -service uninstall                          # Windows, from an admin terminal
   ```

4. Start your build (`gui/build/bin/`) and install the service from it when the app asks. Your settings, accounts and keyring entries are kept.

Self-built apps link to the exchanges' regular sign-up pages rather than our referral links. Everything else behaves the same as the official build.

## Building

Requirements: Go (version in `go.mod`), and for the desktop app Node.js 20+, npm 10+ and the [Wails](https://wails.io) CLI.

Build only the service binary:

```bash
go build -o copy-service ./cmd/copy-service
```

### Build the Wails app

From the `gui` folder:

```bash
wails build
```

This creates a production build of the desktop app. Wails automatically runs `npm ci` and `npm run build` in the `frontend/` directory before compiling Go, so the embedded assets are always up to date. The version created like this will have as Go version string `develop`. If you care about versioning, see further below.

### Build on Linux

If your Linux distribution does not provide `webkit2gtk-4.0` (for example, Ubuntu 24.04 or 26.04), run Wails with the `webkit2_41` build tag:

```bash
wails build -tags webkit2_41
```

### Desktop app versioning (miroir-gui)

Versions are maintained manually (for example `0.1`, `0.2`, `1.0`). Version management is centralized in `gui/wails.json` → `info.productVersion`. The value there is written into the macOS bundle (`CFBundleShortVersionString` / `CFBundleVersion` in `build/darwin/Info.plist` via Wails) and is what Finder and **Get Info** show. Apple expects a period-separated numeric form (`x`, `x.y`, or `x.y.z`); avoid a `v` prefix here.

For a versioned local build, use the `wails build -ldflags ...` command below.

```bash
cd gui
V="$(python3 -c "import json; print(json.load(open('wails.json'))['info']['productVersion'])")"
wails build -ldflags "-X github.com/getmiroir/trading-service/internal/version.Version=${V} -X github.com/getmiroir/trading-service/internal.BinanceReferralURL=https://www.binance.com/activity/referral-entry/CPA?ref=CPA_00XXKO76WB -X github.com/getmiroir/trading-service/internal.BybitReferralURL=https://www.bybit.com/en/invite/?ref=6OXQKNL"
```

#### Exchange API URL overrides (build-time)

Exchange API endpoints and websocket order routing are not configurable in `settings.yaml`. URLs and websocket usage are resolved at runtime from defaults in each exchange client, overridable at build time via `-ldflags` (same pattern as referral URLs). Demo endpoints are selected by `is_demo` on each account and are not overridable. Bybit testnet is selected per account via `is_testnet`, which is only available in develop builds through the desktop GUI.

Optional ldflags (omit for official exchange endpoints and default HTTP routing):

```bash
-X github.com/getmiroir/trading-service/internal/clients.BinanceFuturesBaseURL=https://...
-X github.com/getmiroir/trading-service/internal/clients.BinanceFuturesWebsocketURL=wss://...
-X github.com/getmiroir/trading-service/internal/clients.BinanceSpotBaseURL=https://...
-X github.com/getmiroir/trading-service/internal/clients.BinanceSpotAccountBaseURL=https://...
-X github.com/getmiroir/trading-service/internal/clients.BybitBaseURL=https://...
-X github.com/getmiroir/trading-service/internal/clients.BybitWebsocketURL=wss://...
-X github.com/getmiroir/trading-service/internal/clients.BinanceUseWebsocket=true
-X github.com/getmiroir/trading-service/internal/clients.BybitUseWebsocket=true
```

## Running

The desktop app is the usual entry point: it handles login, exchange accounts, and installing and controlling the background service.

The service binary can also be run directly:

```bash
./copy-service
```

Flags:

- `-service install|uninstall|start|stop|restart|status`: control the OS service
- `-version`: print the version and exit

## Configuration

All data lives in one directory:

- **Linux / macOS**: `~/.miroir` (falls back to the executable's directory if the home directory cannot be determined)
- **Windows**: `%PROGRAMDATA%\Miroir`

It contains the settings file `settings.yaml`, the logs and the service health database. On first run, if `settings.yaml` does not exist, it is created with default values.

Environment overrides:

- `WAILS_APP_DATA_DIR`: use another data directory
- `WAILS_APP_SETTINGS_FILE`: use another settings file (the logs and health database stay in the data directory)

To generate a reference of every setting:

```bash
go run ./cmd/settings-doc -o settings.md
```

### Secrets

Each secret in `settings.yaml` (exchange API keys, the Hyperliquid private key, the Control API key) declares where its value comes from, so the value itself does not have to be in the file. Use one of:

- `from_keyring`: entry name in the system keyring
- `from_env_var`: environment variable name
- `from_file`: path to a file containing the value
- `from_yaml`: `path` to a YAML file and `entry` (list of keys) inside it
- `raw`: the value itself (not recommended)

For example:

```yaml
key:
  from_keyring: miroir_bybit_api_key
```

The keyring service name is `miroir` (GNOME Keyring / KWallet through the Secret Service API on Linux, Keychain on macOS). On Windows, secrets are stored DPAPI-encrypted in `%PROGRAMDATA%\Miroir\secrets`.

### Adding keyring entries on Linux

Use `secret-tool` (from the `libsecret-tools` package, e.g. `apt install libsecret-tools`), and paste the value when prompted:

```bash
secret-tool store --label="Miroir Bybit API key" service miroir username miroir_bybit_api_key
secret-tool store --label="Miroir Bybit API secret" service miroir username miroir_bybit_api_secret
```

Use the same entry names as in your settings. To list or delete entries, use `secret-tool search service miroir` and `secret-tool clear service miroir username <entry_name>`.

## Development

### GUI frontend

The GUI frontend lives at `gui/frontend` and is built with React + Vite.

Frontend only:

```bash
cd gui/frontend
npm install
npm run dev
```

This starts the Vite dev server for the GUI app.

Full Wails app (Go backend + frontend with Wails bindings and hot reload), from `gui`:

```bash
wails dev
```

On Linux without `webkit2gtk-4.0` (see [Build on Linux](#build-on-linux)), use `wails dev -tags webkit2_41`. On Windows, run `wails dev` from an admin PowerShell.

### Quality checks

Go (formats, then builds the service and the GUI):

```bash
bin/type
```

From `gui/frontend`:

```bash
npm run build
npm run type-check
npm run lint-check
npm run format-check
```

Use `npm run lint` and `npm run format` to apply automatic fixes.

### Tests

Unit tests live next to the code (`go test ./internal/...`). End-to-end tests that need exchange API keys are described in [tests/README.md](tests/README.md).

## Architecture

### Overview

- The **desktop app** writes exchange accounts to `settings.yaml` and their secrets to the OS keyring, and controls the background service through the OS service manager.
- The **background service** reads the same settings and secrets at startup and runs the copy loop. It writes a heartbeat to the health database, which the app reads to show whether the service is alive.

### Binaries

| Binary | Built from | Role |
| --- | --- | --- |
| `miroir-gui` | `gui/` | Desktop app. The installed background service runs this same binary with `-run-as-service` (systemd user unit, launchd or Windows service). |
| `copy-service` | `cmd/copy-service` | Headless service only, without the desktop app (used by the Docker image). Validates all accounts at startup. |
| `settings-doc` | `cmd/settings-doc` | Generates the settings reference. |

### Copy loop

Every `loop_sleep_seconds`, the service:

1. **Fetches the copy targets** from the Control API: which trader each account copies, and with which settings.
2. **Syncs each account** concurrently: reads the account's state from its exchange and the trader's state from Hyperliquid, gets the normalization ratio, and checks the account's state. All copied positions are closed if equity is below the minimum, if account or trader data can't be read, or if the Control API flags the copy for an emergency close or blocks the app version.
3. **Syncs each symbol** concurrently: computes the target position size, places the orders to reach it (split into fractions when large), and places or readjusts the stop loss and take profit.
4. **Queues the metrics** for the sync. They are sent to the Control API in batches by a background worker.

### Packages

| Package | Responsibility |
| --- | --- |
| `internal/runners` | Copy loop, account sync and symbol sync |
| `internal/clients` | API clients: Binance, Bybit, Hyperliquid (source and exchange), Control API, symbol mapping, rate limiting, websockets |
| `internal/exchange` | Exchange account and API key validation |
| `internal/settings` | Settings loading, defaults, migration and secret sources; `manager/` handles account changes from the app |
| `internal/keyring` | OS keyring access (DPAPI-encrypted store on Windows) |
| `internal/svc` | OS service install, control and status |
| `internal/health` | Service heartbeat and installed version (SQLite) |
| `internal/metrics` | Metric types and the batching queue |
| `internal/gui` | Desktop app backend: login, deep links, service control, updates |
| `internal/logging` | Logging, file rotation and secret redaction |
| `internal/platform` | Data directory and file locations |
| `internal/version` | Build version, platform and version comparison |
| `internal/utils` | Shared helpers |
| `gui/frontend` | Desktop app UI (React + Vite) |

## Docker

Build the Docker image:

```bash
docker build -t miroir-trading-service:latest -f dockerfile .
```

Run the container (the image reads its settings from `/app/settings.yaml`):

```bash
docker run -v /path/to/settings.yaml:/app/settings.yaml miroir-trading-service:latest
```

There is no system keyring in the container, so use `from_env_var` or `from_file` secrets.

## Disclaimer

This is trading software. It places real orders on your exchange accounts. Use it at your own risk: it is provided without warranty (see [LICENSE.md](LICENSE.md)), it is not financial advice, and you are responsible for your API keys, your settings and any resulting positions or losses.

## License

[EUPL-1.2](LICENSE.md)
