## Live Trading

### Step 1 — Create a Bybit Account

Go to [bybit.com](https://www.bybit.com) and register for an account if you don't have one already.

> **Note:** API key creation is only available via the Bybit **website** — it cannot be done through the Bybit app. Additionally, new accounts must wait **48 hours** after registration before they can create an API key.

### Step 2 — Enable Two-Factor Authentication (2FA)

Before creating an API key, make sure Google Authenticator 2FA is enabled on your account. This is required to complete the API key creation process.

Go to **[Profile icon] → [Account & Security] → [Google Authenticator]** and follow the setup steps if you haven't done so already.

### Step 3 — (Optional) Set Up a Sub-Account

If you want to trade from a dedicated sub-account rather than your main account:

1. Click the **profile icon** in the top-right corner → **[Subaccount]**.
2. Click **[Create Subaccount]** → **[Standard Subaccount]**.
3. Enter a nickname (5–20 alphanumeric characters) and complete the verification.
4. Once created, switch into the sub-account using the **Account Switch** feature before proceeding.

> **Note:** Sub-accounts are created in Unified Trading Account (UTA) mode by default, which combines Spot and Derivatives in a single account.

If you are using your main account, skip this step.

### Step 4 — Create the API Key

1. Click the **profile icon** in the top-right corner → **[API]**.  
   Or navigate directly to: [bybit.com/app/user/api-management](https://www.bybit.com/app/user/api-management)
2. Click **[Create New Key]**.
3. Select **System-generated API Key**.
4. Choose **API Transaction** as the key type (required for trading).
5. Give your key a clear, descriptive name (e.g., `MyApp_Trading`).

### Step 5 — Configure API Permissions

In the permissions panel, enable the following:

| Permission      | Setting                                            |
| --------------- | -------------------------------------------------- |
| Read-Write      | ✅ Enable                                          |
| Unified Trading | ✅ Enable (covers both Spot and Futures under UTA) |
| Orders          | ✅ Enable                                          |
| Positions       | ✅ Enable                                          |

> ⚠️ Do **not** enable withdrawal permissions. For added security, it is strongly recommended to restrict the key to trusted IP addresses.

### Step 6 — Confirm and Save Your Keys

1. Click **[Submit]**.
2. Enter your **Google Authenticator 2FA code** to confirm.
3. Your **API Key** and **Secret Key** will be displayed.

> ⚠️ The Secret Key is shown only once. Copy and store it securely before closing the page. If lost, you will need to delete the key and create a new one.

### Step 7 — Add the Account to the Application

Open the desktop application and follow the in-app instructions to add your Bybit account using the API Key and Secret Key you just created.

## Margin and Position Mode

If account setup fails with a margin or position mode error, configure these settings on Bybit, then try saving again.

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
