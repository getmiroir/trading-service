import { ModalBackdrop, type ModalBackdropProps } from './ModalBackdrop';
import { ModalPanel, type ModalPanelProps } from './ModalPanel';

type ModalProps = ModalBackdropProps & ModalPanelProps;

export const Modal = ({
	onClose,
	closeOnEscape,
	closeOnBackdropClick,
	scrollable,
	children,
	...panelProps
}: ModalProps) => (
	<ModalBackdrop
		onClose={onClose}
		closeOnEscape={closeOnEscape}
		closeOnBackdropClick={closeOnBackdropClick}
		scrollable={scrollable}
	>
		<ModalPanel {...panelProps} onClose={onClose}>
			{children}
		</ModalPanel>
	</ModalBackdrop>
);
