import { AccountApiKeyField, AccountApiSecretField } from './FormElements';
import { FormSection } from './FormSection';

type AccountOverlayExchangeCredentialsProps = {
	isEdit: boolean;
	apiKey: string;
	apiSecret: string;
	onSetApiKey: (value: string) => void;
	onSetApiSecret: (value: string) => void;
};

export const AccountOverlayExchangeCredentials = ({
	isEdit,
	apiKey,
	apiSecret,
	onSetApiKey,
	onSetApiSecret,
}: AccountOverlayExchangeCredentialsProps) => (
	<FormSection title='API credentials'>
		<AccountApiKeyField
			value={apiKey}
			onChange={onSetApiKey}
			placeholder={isEdit ? 'Enter new API key' : 'Enter API key'}
		/>
		<AccountApiSecretField
			value={apiSecret}
			onChange={onSetApiSecret}
			placeholder={isEdit ? 'Enter new API secret' : 'Enter API secret'}
		/>
	</FormSection>
);
