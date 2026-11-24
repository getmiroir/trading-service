// Mirrors subsettings.ClientType in the Go settings module. Kept in sync by hand,
// like the other Go types declared in vite-env.d.ts. Use these constants instead
// of bare string literals so the values live in a single place.
export const Exchange = {
	Binance: 'binance',
	Bybit: 'bybit',
	Hyperliquid: 'hyperliquid',
} as const;

export type ExchangeType = (typeof Exchange)[keyof typeof Exchange];

export type BybitRegion = '' | 'tr' | 'kz';

export const bybitRegionLabel = (region: BybitRegion) => {
	if (region === 'tr') {
		return 'Turkey';
	}
	if (region === 'kz') {
		return 'Kazakhstan';
	}
	return 'Global';
};

export type OverlayMode = 'closed' | 'picker' | 'addNotice' | 'addForm' | 'editForm';

export const exchangeLabel = (exchange: ExchangeType) => {
	if (exchange === Exchange.Binance) {
		return 'Binance';
	}
	if (exchange === Exchange.Bybit) {
		return 'Bybit';
	}
	if (exchange === Exchange.Hyperliquid) {
		return 'Hyperliquid';
	}
	throw new Error(`Unknown exchange: ${exchange}`);
};
