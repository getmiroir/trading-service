import { useId } from 'react';

import { Modal } from './Modal';
import { ModalActions } from './ModalActions';
import { ModalButton } from './ModalButton';
import { type ReportResult } from './ReportModal.utils';
import { CheckCircleIcon } from './icons/CheckCircleIcon';
import { XCircleIcon } from './icons/XCircleIcon';

interface ReportModalResultProps {
	result: ReportResult;
	onClose: () => void;
}

export const ReportModalResult = ({ result, onClose }: ReportModalResultProps) => {
	const messageId = useId();

	return (
		<Modal
			tone={result.success ? 'primary' : 'danger'}
			icon={result.success ? <CheckCircleIcon /> : <XCircleIcon />}
			title={result.success ? 'Report sent' : 'Report not sent'}
			descriptionId={messageId}
			onClose={onClose}
		>
			<p id={messageId} className='modal__message'>
				{result.message}
			</p>
			<ModalActions>
				<ModalButton variant='secondary' onClick={onClose} autoFocus>
					Close
				</ModalButton>
			</ModalActions>
		</Modal>
	);
};
