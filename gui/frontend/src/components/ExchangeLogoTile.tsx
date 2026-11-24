import { type ExchangeType } from '../types';
import { ExchangeLogo } from './ExchangeLogo';

type ExchangeLogoTileProps = {
	exchange: ExchangeType;
};

export const ExchangeLogoTile = ({ exchange }: ExchangeLogoTileProps) => (
	<span className='exchange-logo-tile' aria-hidden='true'>
		<ExchangeLogo exchange={exchange} />
	</span>
);
