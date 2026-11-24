import { type ReactNode, useId } from 'react';

import { ModalHeader } from './ModalHeader';

export type ModalTone = 'primary' | 'warning' | 'danger';
export type ModalSize = 'sm' | 'md' | 'lg';

export type ModalPanelProps = {
	title?: string;
	icon?: ReactNode;
	leading?: ReactNode;
	subtitle?: ReactNode;
	tone?: ModalTone;
	size?: ModalSize;
	role?: 'dialog' | 'alertdialog';
	descriptionId?: string;
	onClose?: () => void;
	closeDisabled?: boolean;
	children: ReactNode;
};

export const ModalPanel = ({
	title,
	icon,
	leading,
	subtitle,
	tone = 'primary',
	size = 'sm',
	role = 'dialog',
	descriptionId,
	onClose,
	closeDisabled,
	children,
}: ModalPanelProps) => {
	const titleId = useId();
	const layoutClassName = icon ? ' modal--centered' : '';

	return (
		<div
			className={`modal modal--${tone} modal--${size}${layoutClassName}`}
			role={role}
			aria-modal='true'
			aria-labelledby={title ? titleId : undefined}
			aria-describedby={descriptionId}
		>
			<ModalHeader
				titleId={titleId}
				title={title}
				icon={icon}
				leading={leading}
				subtitle={subtitle}
				onClose={onClose}
				closeDisabled={closeDisabled}
			/>
			{children}
		</div>
	);
};
