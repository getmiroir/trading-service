# How to get your Hyperliquid API Credentials

## How to start with Hyperliquid

If you haven't used Hyperliquid yet, we recommend following [this tutorial](https://hyperliquid.gitbook.io/hyperliquid-docs/onboarding/how-to-start-trading).

## Generate an API wallet in Hyperliquid

### In [Hyperliquid](https://app.hyperliquid.xyz/trade)

1. Connect using your wallet
2. Go to **[More]** > **[API]**
3. Enter a name for the API wallet, and generate an address for it (click **[Generate]**). Then click **[Authorize API Wallet]**.
4. **COPY THE PRIVATE KEY** visible in the pop-up. It won't appear again.
5. Choose the number of days of validity for your API Wallet. We recommend using the maximum.
6. Confirm the authorization by clicking **[Authorize]** and signing the operation in your wallet.

### In the app

Enter:

- the copied private key
- your wallet address in Hyperliquid _Not the address of the API wallet_
- _(Optional)_ A subaccount address if you are using one
- _(Optional)_ Minimum equity needed on the account for the application to mirror positions

## Sub-account support

If you want to trade on behalf of a sub-account, enter its address in the **Subaccount Address** field. The main wallet's private key still signs all trading actions, but they take place in the sub-account.

## Security

Your private key is stored encrypted in your system keyring and never leaves your device.
