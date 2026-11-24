## Live Trading

### Step 1 — Create a Binance Account

Go to [binance.com](https://www.binance.com) and register for an account if you don't have one already.

### Step 2 — Verify Futures Eligibility

Before going further, confirm that **Futures trading is available in your country**. Binance restricts Futures access in certain regions. If Futures is not available to you, you will not be able to use the application for live trading and can stop here.

### Step 3 — Complete Identity Verification (KYC)

Binance requires identity verification before you can create a trading API key.

1. Go to your profile and open **Identification**.
2. Follow the prompts to submit your ID documents.
3. Wait for approval (usually within a few minutes to a few hours).

You must also make a deposit of any amount to your Spot Wallet to activate your account.

> **Note:** Two-factor authentication (2FA) must also be enabled on your account before you can create an API key.

### Step 4 — Enable Your Futures Account

Once verified, activate Futures trading:

1. Go to **[Trade] → [Futures]**.
2. Follow the on-screen instructions to open your Futures account.

> ⚠️ **Important:** You must enable your Futures account **before** creating your API key. An API key created prior to Futures activation will not have Futures permissions, and you will need to create a new one.

### Step 5 — (Optional) Set Up a Sub-Account

If you want to trade from a dedicated sub-account rather than your main account:

1. Go to **[Profile] → [Account] → [Sub-Account]** (or navigate to **Settings → Sub-Accounts**).
2. Create a new sub-account and complete any required verification.
3. Switch into the sub-account before proceeding to the next step.

If you are using your main account, skip this step.

### Step 6 — Create the API Key

1. Click your **profile icon** → **[Account]** → **[API Management]**.
2. Click **[Create API]**.
3. Select **System Generated** as the key type.
4. Give your key a clear, descriptive name (e.g., `MyApp_Trading`).
5. Complete the 2FA verification prompt.

### Step 7 — Configure API Permissions

Once the key is created, open its settings and enable the following permissions:

| Permission                   | Required |
| ---------------------------- | -------- |
| Enable Reading               | ✅ Yes   |
| Enable Spot & Margin Trading | ✅ Yes   |
| Enable Futures               | ✅ Yes   |

> ⚠️ **Important — IP Restrictions:** By default, Binance only allows **Read** permission on unrestricted API keys. To enable trading permissions, you must either restrict the key to trusted IP addresses (recommended) or disable the default security controls. Do **not** enable withdrawal permissions.

### Step 8 — Save Your Keys

After creation, you will be shown your **API Key** and **Secret Key**.

> ⚠️ The Secret Key is only displayed once. Copy and store it securely before closing the page. If lost, you will need to delete the key and create a new one.

### Step 9 — Add the Account to the Application

Open the desktop application and follow the in-app instructions to add your Binance account using the API Key and Secret Key you just created.

## Margin and Position Mode

If account setup fails with a margin or position mode error, configure these settings on Binance, then try saving again.

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
