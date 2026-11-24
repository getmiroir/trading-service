import { useId } from 'react';

import { Modal } from './Modal';
import { ModalActions } from './ModalActions';
import { ModalButton } from './ModalButton';
import { ExclamationTriangleIcon } from './icons/ExclamationTriangleIcon';

type ConfirmModalTone = 'danger' | 'warning';

interface ConfirmModalProps {
	title?: string;
	message: string;
	details?: string[];
	tone?: ConfirmModalTone;
	confirmLabel?: string;
	cancelLabel?: string;
	onConfirm: () => void;
	onCancel: () => void;
}

export const ConfirmModal = ({
	title,
	message,
	details,
	tone = 'danger',
	confirmLabel = 'Remove',
	cancelLabel = 'Cancel',
	onConfirm,
	onCancel,
}: ConfirmModalProps) => {
	const messageId = useId();

	return (
		<Modal
			role='alertdialog'
			tone={tone}
			icon={<ExclamationTriangleIcon />}
			title={title}
			descriptionId={messageId}
			onClose={onCancel}
		>
			<p id={messageId} className='modal__message'>
				{message}
			</p>
			{details && details.length > 0 && (
				<ul className='modal__details'>
					{details.map((detail) => (
						<li key={detail} className='modal__detail'>
							{detail}
						</li>
					))}
				</ul>
			)}
			<ModalActions>
				<ModalButton variant='secondary' onClick={onCancel} autoFocus>
					{cancelLabel}
				</ModalButton>
				<ModalButton variant='danger' onClick={onConfirm}>
					{confirmLabel}
				</ModalButton>
			</ModalActions>
		</Modal>
	);
};
