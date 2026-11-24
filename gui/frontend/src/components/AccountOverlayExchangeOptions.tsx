import { BybitRegion, Exchange, ExchangeType } from '../types';
import { AccountOverlayBybitRegionSection } from './AccountOverlayBybitRegionSection';
import { FormSection } from './FormSection';
import { Switch } from './Switch';

type AccountOverlayExchangeOptionsProps = {
	selectedExchange?: ExchangeType;
	editAccountID?: string;
	isDevelopMode: boolean;
	demo: boolean;
	testnet: boolean;
	bybitRegion: BybitRegion;
	bybitRegionEnabled: boolean;
	onSetDemo: (value: boolean) => void;
	onSetTestnet: (value: boolean) => void;
	onSetBybitRegion: (value: BybitRegion) => void;
	onSetBybitRegionEnabled: (value: boolean) => void;
};

export const AccountOverlayExchangeOptions = ({
	selectedExchange,
	editAccountID,
	isDevelopMode,
	demo,
	testnet,
	bybitRegion,
	bybitRegionEnabled,
	onSetDemo,
	onSetTestnet,
	onSetBybitRegion,
	onSetBybitRegionEnabled,
}: AccountOverlayExchangeOptionsProps) => {
	const isBybit = selectedExchange === Exchange.Bybit;
	const showDemo = !editAccountID;
	const showTestnet = isBybit && isDevelopMode;

	if (!showDemo && !isBybit) return null;

	return (
		<FormSection title='Options'>
			{showDemo && (
				<Switch
					id='demoSwitch'
					label='Demo account'
					checked={demo}
					disabled={testnet}
					onChange={onSetDemo}
				/>
			)}
			{isBybit && (
				<AccountOverlayBybitRegionSection
					key={editAccountID ?? 'add'}
					bybitRegion={bybitRegion}
					bybitRegionEnabled={bybitRegionEnabled}
					onSetBybitRegion={onSetBybitRegion}
					onSetBybitRegionEnabled={onSetBybitRegionEnabled}
				/>
			)}
			{showTestnet && (
				<Switch
					id='testnetSwitch'
					label='Testnet account'
					description='Develop only'
					checked={testnet}
					onChange={onSetTestnet}
				/>
			)}
		</FormSection>
	);
};
