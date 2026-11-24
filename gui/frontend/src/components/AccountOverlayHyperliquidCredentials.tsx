import { AccountApiSecretField, AccountWalletAddressField } from './FormElements';
import { FormSection } from './FormSection';

type AccountOverlayHyperliquidCredentialsProps = {
	isEdit: boolean;
	privateKey: string;
	masterWalletAddress: string;
	subaccountAddress: string;
	onSetPrivateKey: (value: string) => void;
	onSetMasterWalletAddress: (value: string) => void;
	onSetSubaccountAddress: (value: string) => void;
};

export const AccountOverlayHyperliquidCredentials = ({
	isEdit,
	privateKey,
	masterWalletAddress,
	subaccountAddress,
	onSetPrivateKey,
	onSetMasterWalletAddress,
	onSetSubaccountAddress,
}: AccountOverlayHyperliquidCredentialsProps) => (
	<FormSection title='API credentials'>
		<AccountApiSecretField
			value={privateKey}
			onChange={onSetPrivateKey}
			label='Private API key'
			placeholder={isEdit ? 'Enter new private API key' : 'Hex ECDSA private API key (0x...)'}
		/>
		<AccountWalletAddressField
			id='masterWalletAddressInput'
			label='Master wallet address'
			value={masterWalletAddress}
			onChange={onSetMasterWalletAddress}
		/>
		<AccountWalletAddressField
			id='subaccountAddressInput'
			label='Subaccount address'
			value={subaccountAddress}
			onChange={onSetSubaccountAddress}
			optional
		/>
	</FormSection>
);
