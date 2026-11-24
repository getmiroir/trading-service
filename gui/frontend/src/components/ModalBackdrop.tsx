import { type ReactNode } from 'react';

import { useEscapeKey } from '../hooks/useEscapeKey';
import './Modal.css';

export type ModalBackdropProps = {
	onClose: () => void;
	closeOnEscape?: boolean;
	closeOnBackdropClick?: boolean;
	scrollable?: boolean;
	children: ReactNode;
};

export const ModalBackdrop = ({
	onClose,
	closeOnEscape = true,
	closeOnBackdropClick = true,
	scrollable = false,
	children,
}: ModalBackdropProps) => {
	useEscapeKey(onClose, closeOnEscape);

	return (
		<div
			className={scrollable ? 'modal-backdrop modal-backdrop--scroll' : 'modal-backdrop'}
			onClick={(event) => {
				if (closeOnBackdropClick && event.target === event.currentTarget) {
					onClose();
				}
			}}
		>
			{children}
		</div>
	);
};
