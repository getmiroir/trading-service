import { useId } from 'react';

import { buildAccountDeepLink } from './AccountRemovalBlockedModal.utils';
import { Modal } from './Modal';
import { ModalActions } from './ModalActions';
import { ModalButton } from './ModalButton';
import { ExclamationTriangleIcon } from './icons/ExclamationTriangleIcon';

type AccountRemovalBlockedModalProps = {
	accountID: string;
	accountName: string;
	loginURL?: string;
	onDismiss: () => void;
};

export const AccountRemovalBlockedModal = ({
	accountID,
	accountName,
	loginURL,
	onDismiss,
}: AccountRemovalBlockedModalProps) => {
	const messageId = useId();
	const deepLink = buildAccountDeepLink(loginURL, accountID);

	const handleOpenWebapp = (url: string) => {
		void window.runtime.BrowserOpenURL(url);
		onDismiss();
	};

	return (
		<Modal
			role='alertdialog'
			tone='warning'
			icon={<ExclamationTriangleIcon />}
			title={`"${accountName}" is still mirroring`}
			descriptionId={messageId}
			onClose={onDismiss}
		>
			<p id={messageId} className='modal__message'>
				Stop mirroring for this account on the webapp first, then try removing it again.
			</p>
			<ModalActions>
				<ModalButton variant='secondary' onClick={onDismiss} autoFocus={!deepLink}>
					OK
				</ModalButton>
				{deepLink && (
					<ModalButton
						variant='primary'
						onClick={() => handleOpenWebapp(deepLink)}
						autoFocus
					>
						Open in webapp
					</ModalButton>
				)}
			</ModalActions>
		</Modal>
	);
};
