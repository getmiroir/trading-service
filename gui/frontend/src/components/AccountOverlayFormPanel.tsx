import { type ReactNode } from 'react';

import { type ExchangeType } from '../types';
import { getAccountFormTitle } from './AccountOverlayFormPanel.utils';
import { AccountOverlayFormSubtitle } from './AccountOverlayFormSubtitle';
import { ExchangeLogoTile } from './ExchangeLogoTile';
import { ModalPanel } from './ModalPanel';

type AccountOverlayFormPanelProps = {
	exchange?: ExchangeType;
	editAccountID?: string;
	editAccountName?: string;
	referralURL?: string;
	formSubmitting: boolean;
	onClose: () => void;
	children: ReactNode;
};

export const AccountOverlayFormPanel = ({
	exchange,
	editAccountID,
	editAccountName,
	referralURL,
	formSubmitting,
	onClose,
	children,
}: AccountOverlayFormPanelProps) => {
	const isEdit = !!editAccountID;
	const subtitleName = isEdit ? editAccountName : undefined;
	const subtitleReferral = isEdit ? undefined : referralURL;
	const hasSubtitle = !!exchange && (!!subtitleName || !!subtitleReferral);

	return (
		<ModalPanel
			size='md'
			title={getAccountFormTitle(exchange, isEdit)}
			leading={exchange && <ExchangeLogoTile exchange={exchange} />}
			subtitle={
				hasSubtitle &&
				exchange && (
					<AccountOverlayFormSubtitle
						exchange={exchange}
						editAccountName={subtitleName}
						referralURL={subtitleReferral}
					/>
				)
			}
			onClose={onClose}
			closeDisabled={formSubmitting}
		>
			{children}
		</ModalPanel>
	);
};
