import { FormSection } from './FormSection';
import { Switch } from './Switch';

type AccountOverlayHyperliquidOptionsProps = {
	isDevelopMode: boolean;
	isTestNet: boolean;
	onSetIsTestNet: (value: boolean) => void;
};

export const AccountOverlayHyperliquidOptions = ({
	isDevelopMode,
	isTestNet,
	onSetIsTestNet,
}: AccountOverlayHyperliquidOptionsProps) => {
	if (!isDevelopMode) return null;

	return (
		<FormSection title='Options'>
			<Switch
				id='testnetSwitch'
				label='Testnet account'
				description='Develop only'
				checked={isTestNet}
				onChange={onSetIsTestNet}
			/>
		</FormSection>
	);
};
