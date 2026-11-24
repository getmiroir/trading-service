import { type ExchangeType, exchangeLabel } from '../types';
import { ExchangeLogoTile } from './ExchangeLogoTile';

type AccountOverlayPickerOptionProps = {
	exchange: ExchangeType;
	description: string;
	onPick: (exchange: ExchangeType) => void;
};

export const AccountOverlayPickerOption = ({
	exchange,
	description,
	onPick,
}: AccountOverlayPickerOptionProps) => (
	<button type='button' className='exchange-btn' onClick={() => onPick(exchange)}>
		<ExchangeLogoTile exchange={exchange} />
		<span className='exchange-btn__text'>
			<span className='exchange-label'>{exchangeLabel(exchange)}</span>
			<span className='exchange-desc'>{description}</span>
		</span>
		<span className='exchange-btn__chevron' aria-hidden='true'>
			›
		</span>
	</button>
);
