import { Exchange, type ExchangeType } from '../types';
import { AccountOverlayPickerOption } from './AccountOverlayPickerOption';
import { Modal } from './Modal';

const EXCHANGE_OPTIONS: { exchange: ExchangeType; description: string }[] = [
	{ exchange: Exchange.Binance, description: 'Futures trading via Binance API' },
	{ exchange: Exchange.Bybit, description: 'Contract trading via Bybit API' },
	{ exchange: Exchange.Hyperliquid, description: 'Perpetuals trading via API wallet' },
];

type AccountOverlayPickerProps = {
	enabledClients: ExchangeType[];
	onPickExchange: (exchange: ExchangeType) => void;
	onClose: () => void;
};

export const AccountOverlayPicker = ({
	enabledClients,
	onPickExchange,
	onClose,
}: AccountOverlayPickerProps) => (
	<Modal size='md' title='Add account' onClose={onClose}>
		<div className='exchange-picker'>
			{EXCHANGE_OPTIONS.filter(({ exchange }) => enabledClients.includes(exchange)).map(
				({ exchange, description }) => (
					<AccountOverlayPickerOption
						key={exchange}
						exchange={exchange}
						description={description}
						onPick={onPickExchange}
					/>
				),
			)}
		</div>
	</Modal>
);
