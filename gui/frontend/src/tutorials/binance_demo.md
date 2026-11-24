## Demo Trading

Binance Demo Trading is a simulated environment using virtual funds. It mirrors live trading on both Spot and Futures markets and does not require identity verification or a funded account.

> **Note:** Demo API keys do not appear to have an expiration date, unlike production keys which should be rotated regularly as a security best practice.

> ⚠️ **Sub-account + Demo Trading:** Binance does not officially document demo trading support on sub-accounts. Unlike Bybit, which explicitly offers an independent demo environment per sub-account, Binance's demo trading is tied to the main account you authenticate with. If you need to demo trade under a specific sub-account context, this is not a supported or documented workflow on Binance at this time.

### Step 1 — Create a Binance Account

Go to [binance.com](https://www.binance.com) and register if you don't have an account. No KYC or deposit is required for demo access.

### Step 2 — Access Demo Trading

Log in, then navigate to **[Trade] → [Demo Trading]** from the top navigation bar.

Alternatively, you can go directly to [demo.binance.com](https://demo.binance.com).

### Step 3 — Create a Demo API Key

Once inside the Demo Trading environment:

1. Click the **user/account icon** in the top-right corner.
2. Go to **[API Management]** (or navigate directly via the link shown in the Demo interface).
3. Click **[Create API]** and give your key a name.

### Step 4 — Configure API Permissions

Enable the following permissions on your demo API key:

| Permission                   | Required |
| ---------------------------- | -------- |
| Enable Reading               | ✅ Yes   |
| Enable Spot & Margin Trading | ✅ Yes   |
| Enable Futures               | ✅ Yes   |

### Step 5 — Save Your Keys

Copy both the **API Key** and **Secret Key** immediately — the Secret Key will not be shown again.

### Step 6 — Add the Account to the Application

Open the desktop application and add your Binance Demo account using the demo API Key and Secret Key.

## Margin and Position Mode

If account setup fails with a margin or position mode error, configure these settings on Binance Demo Trading, then try saving again.

The application requires **One-Way** position mode (account-wide) and **Isolated** margin mode on each contract you trade.

> **Note:** When you save your account, the application will try to apply these settings automatically via the API. This can fail if you have open positions or orders — use the steps below to configure them manually in the exchange UI.

> **Before you start:** Close all open positions and cancel all active orders for any contract you want to change.

> **Note:** On Binance, margin mode is set per contract. Position mode is account-wide across all contracts.

### Step 1 — Set Isolated Margin Mode (per contract)

1. Go to the Futures trading interface (USDⓈ-M).
2. Select the contract you want, e.g. BTCUSDT.
3. In the top-right area of the trading panel, click where it shows **[Cross]**.
4. A popup will appear — select **Isolated**.
5. Click **Confirm**.
6. Repeat this for every contract you want in Isolated mode — it does not apply globally.

### Step 2 — Set One-Way Position Mode (account-wide)

**On desktop:**

1. On the Futures trading page, click the **Settings** icon (top right).
2. Click **Preferences**.
3. Click **Position Mode**.
4. Select **One-Way Mode**.
5. Confirm.

**On mobile:**

1. Open the Binance app.
2. Tap **Futures → USDⓈ-M**.
3. Tap the icon in the top right corner.
4. Tap **Preferences**.
5. Tap **Position Mode Setting**.
6. Select **One-Way Mode**.
7. Confirm.

### Key rules to remember

- Default is Cross Margin + One-Way mode on all contracts.
- Margin mode is per contract — you must switch each one individually.
- Position mode (One-Way/Hedge) is account-wide — changing it affects all contracts.
- You cannot change margin mode if there are open orders or open positions on that contract.
- You cannot change position mode if there are any open orders or positions anywhere on the account.
- In One-Way mode, opening the opposite direction will reduce or close your current position.
- Hedge mode allows simultaneous long and short on the same contract.

### While trading

If the application reports a **margin mode** error while mirroring positions, set **Isolated** margin for each contract being traded in the Futures UI (Step 1). Position mode remains account-wide (Step 2).
