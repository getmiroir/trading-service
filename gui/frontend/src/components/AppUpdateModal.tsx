import { useId } from 'react';

import { Modal } from './Modal';
import { ModalActions } from './ModalActions';
import { ModalButton } from './ModalButton';
import { VersionComparison } from './VersionComparison';
import { DownloadIcon } from './icons/DownloadIcon';

type AppUpdateModalProps = {
	currentVersion: string;
	latestVersion: string;
	onDownload: () => void;
	onDismiss: () => void;
};

export const AppUpdateModal = ({
	currentVersion,
	latestVersion,
	onDownload,
	onDismiss,
}: AppUpdateModalProps) => {
	const messageId = useId();

	return (
		<Modal
			icon={<DownloadIcon />}
			title='Update available'
			descriptionId={messageId}
			onClose={onDismiss}
		>
			<p id={messageId} className='modal__message'>
				A new version of Miroir is available.
			</p>
			<VersionComparison
				rows={[
					{ label: 'Current version', value: currentVersion, variant: 'outgoing' },
					{ label: 'New version', value: latestVersion, variant: 'incoming' },
				]}
			/>
			<ModalActions>
				<ModalButton variant='secondary' onClick={onDismiss}>
					Not now
				</ModalButton>
				<ModalButton variant='primary' onClick={onDownload} autoFocus>
					Download
				</ModalButton>
			</ModalActions>
		</Modal>
	);
};
