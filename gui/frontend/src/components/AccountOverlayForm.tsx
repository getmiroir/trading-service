import { type ReactNode } from 'react';

import { type ExchangeType } from '../types';
import { AccountOverlayFormButtons } from './AccountOverlayFormButtons';
import { AccountOverlayFormPanel } from './AccountOverlayFormPanel';
import { FormAlert } from './FormAlert';
import { AccountMinimumEquityField } from './FormElements';
import { FormSection } from './FormSection';

type AccountOverlayFormProps = {
	exchange?: ExchangeType;
	editAccountID?: string;
	editAccountName?: string;
	referralURL?: string;
	formLoading: boolean;
	formError: string;
	formSubmitting: boolean;
	minimumEquity: string;
	onSetMinimumEquity: (value: string) => void;
	credentials: ReactNode;
	options: ReactNode;
	onClose: () => void;
	onSubmit: () => void;
};

export const AccountOverlayForm = ({
	exchange,
	editAccountID,
	editAccountName,
	referralURL,
	formLoading,
	formError,
	formSubmitting,
	minimumEquity,
	onSetMinimumEquity,
	credentials,
	options,
	onClose,
	onSubmit,
}: AccountOverlayFormProps) => (
	<AccountOverlayFormPanel
		exchange={exchange}
		editAccountID={editAccountID}
		editAccountName={editAccountName}
		referralURL={referralURL}
		formSubmitting={formSubmitting}
		onClose={onClose}
	>
		{formLoading ? (
			<div className='form-loading'>Loading account…</div>
		) : (
			<>
				{credentials}
				<FormSection title='Safety'>
					<AccountMinimumEquityField
						value={minimumEquity}
						onChange={onSetMinimumEquity}
					/>
				</FormSection>
				{options}
			</>
		)}
		<FormAlert message={formError} />
		{!formLoading && (
			<AccountOverlayFormButtons
				editAccountID={editAccountID}
				formSubmitting={formSubmitting}
				onClose={onClose}
				onSubmit={onSubmit}
			/>
		)}
	</AccountOverlayFormPanel>
);
