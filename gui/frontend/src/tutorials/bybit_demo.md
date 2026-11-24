## Demo Trading

Bybit Demo Trading is a simulated environment using virtual funds. It mirrors live trading conditions on both Spot and Derivatives (Futures) markets under a Unified Trading Account, and does not require identity verification.

> **Note:** Your Demo account does not expire as long as your Bybit account is active. However, if you do not access it for **30 days**, the account data will be cleared and reset automatically.

### Step 1 — Create a Bybit Account

Go to [bybit.com](https://www.bybit.com) and register if you don't have an account. No KYC or deposit is required to access Demo Trading.

> Upon first access, your Demo account is automatically credited with **50,000 USDT, 50,000 USDC, 1 BTC, and 1 ETH** in virtual funds.

### Step 2 — (Optional) Switch to a Sub-Account First

Bybit Demo Trading is supported on sub-accounts. Each sub-account has its **own independent demo account**, completely separate from the main account's demo environment. This means you can demo trade on a sub-account without affecting the demo account of your main account or any other sub-account.

If you want to demo trade from a specific sub-account:

1. From your main account, click the **profile icon** → **[Subaccount]** → **[Create Subaccount]** (if not already created — refer to Part 1, Step 3 for full instructions).
2. Switch into the sub-account using the **Account Switch** feature.
3. Once you are logged in as the sub-account, proceed to the next step.

If you are demo trading from your main account, skip this step.

### Step 3 — Switch to Demo Trading

While logged into the account you want to demo trade with (main or sub-account), hover over the **profile icon** in the top-right corner and click **[Demo Trading]** from the dropdown.

You will see a **"Demo Trading"** label in the header confirming you are now in the simulated environment, scoped to the account you switched from.

### Step 4 — Create a Demo API Key

Once inside the Demo Trading environment:

1. Click the **profile icon** (still visible in the top-right corner) → **[API]**.
2. Click **[Create New Key]**.
3. Select **System-generated API Key**.
4. Choose **API Transaction** as the key type.
5. Give your key a descriptive name (e.g., `MyApp_Demo` or `MyApp_SubAccount_Demo`).

> ⚠️ The Demo API key must be created **from within the Demo Trading environment** of the specific account you want to use (main or sub-account). A key created from the live account will not work with the demo endpoints, and a key created from the main account's demo will not work for a sub-account's demo.

### Step 5 — Configure API Permissions

Enable the same permissions as production:

| Permission      | Setting   |
| --------------- | --------- |
| Read-Write      | ✅ Enable |
| Unified Trading | ✅ Enable |
| Orders          | ✅ Enable |
| Positions       | ✅ Enable |

### Step 6 — Confirm and Save Your Keys

1. Click **[Submit]** and complete the 2FA verification.
2. Copy both your **API Key** and **Secret Key** immediately — the Secret Key will not be shown again.

### Step 7 — Add the Account to the Application

Open the desktop application and add your Bybit Demo account using the demo API Key and Secret Key.

## Margin and Position Mode

If account setup fails with a margin or position mode error, configure these settings on Bybit Demo Trading, then try saving again.

The application requires **Isolated Margin** and **One-Way** position mode.

> **Note:** When you save your account, the application will try to apply these settings automatically via the API. This can fail if you have open positions or orders — use the steps below to configure them manually in the exchange UI.

> **Before you start:** Close all open positions and cancel all active orders. You also need no existing borrowings and spot margin trading must be disabled.

> **Note:** On Bybit, margin mode applies to your entire account (not per contract). Position mode applies per symbol (or you can apply it to all USDT Perps at once).

### Step 1 — Set Isolated Margin Mode

1. Go to the Futures trading page (e.g. BTCUSDT Perpetual).
2. In the order panel on the right, tap the **Ellipsis** icon (**···**).
3. Your current margin mode will be shown at the bottom of the screen.
4. Select **Isolated Margin**.
5. Tap **Switch Margin Mode** to confirm.

### Step 2 — Set One-Way Position Mode

1. On the same Futures trading page, tap the **Ellipsis** icon (**···**) again.
2. Go to **Trading Preferences**.
3. You'll see your current position mode.
4. Select **One-Way Mode**.
5. Tap **OK** to confirm.

> **Optional:** enable **Apply to all USDT-Perp** to apply One-Way mode across all USDT Perpetual pairs that have no open orders or positions.

### Key rules to remember

- Default is Cross Margin + 10x leverage + One-Way mode.
- Isolated margin mode is account-wide — it applies to all your futures positions, not just one symbol.
- One-Way mode means opening in the opposite direction will reduce or close your existing position.
- Hedge mode (two directions simultaneously) is only available on USDT Perpetual contracts.
- You cannot switch margin mode if you have open options positions, spot margin orders, or existing borrowings.
