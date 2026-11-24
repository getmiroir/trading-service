const HYPERLIQUID_ADDRESS_REGEX = /0x[0-9a-fA-F]{40}$/;

export const truncateHyperliquidAccountID = (accountID: string): string => {
	const address = accountID.match(HYPERLIQUID_ADDRESS_REGEX)?.[0];
	if (!address) return accountID;
	return `${address.slice(0, 6)}...${address.slice(-4)}`;
};

export const getAccountDisplayName = (accountID: string, labels: Record<string, string>): string =>
	labels[accountID] ?? truncateHyperliquidAccountID(accountID);
