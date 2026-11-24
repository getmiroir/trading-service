import { type ReactNode } from 'react';

type ModalButtonVariant = 'secondary' | 'primary' | 'danger';

type ModalButtonProps = {
	variant: ModalButtonVariant;
	onClick: () => void;
	disabled?: boolean;
	autoFocus?: boolean;
	children: ReactNode;
};

export const ModalButton = ({
	variant,
	onClick,
	disabled,
	autoFocus,
	children,
}: ModalButtonProps) => (
	<button
		type='button'
		className={`modal__btn modal__btn--${variant}`}
		onClick={onClick}
		disabled={disabled}
		autoFocus={autoFocus}
	>
		{children}
	</button>
);
