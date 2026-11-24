import binanceLogo from '../assets/binance_logo.svg';
import bybitLogo from '../assets/bybit_logo.svg';
import hyperliquidLogo from '../assets/hyperliquid_logo.svg';
import { Exchange, type ExchangeType, exchangeLabel } from '../types';

const logoByExchange: Record<ExchangeType, string> = {
	[Exchange.Binance]: binanceLogo,
	[Exchange.Bybit]: bybitLogo,
	[Exchange.Hyperliquid]: hyperliquidLogo,
};

type ExchangeLogoProps = {
	exchange: ExchangeType;
};

export const ExchangeLogo = ({ exchange }: ExchangeLogoProps) => (
	<img
		className='exchange-logo'
		src={logoByExchange[exchange]}
		alt={exchangeLabel(exchange)}
		title={exchangeLabel(exchange)}
	/>
);
