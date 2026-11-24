import { type ReactNode } from 'react';

type ModalHeaderProps = {
	titleId: string;
	title?: string;
	icon?: ReactNode;
	leading?: ReactNode;
	subtitle?: ReactNode;
	onClose?: () => void;
	closeDisabled?: boolean;
};

export const ModalHeader = ({
	titleId,
	title,
	icon,
	leading,
	subtitle,
	onClose,
	closeDisabled,
}: ModalHeaderProps) => {
	if (icon) {
		return (
			<>
				<div className='modal__icon' aria-hidden='true'>
					{icon}
				</div>
				{title && (
					<h3 id={titleId} className='modal__title'>
						{title}
					</h3>
				)}
			</>
		);
	}

	if (!title && !onClose) return null;

	return (
		<div className='modal__header'>
			<div className='modal__heading'>
				{leading && <div className='modal__leading'>{leading}</div>}
				<div className='modal__heading-text'>
					{title && (
						<h3 id={titleId} className='modal__title'>
							{title}
						</h3>
					)}
					{subtitle && <p className='modal__subtitle'>{subtitle}</p>}
				</div>
			</div>
			{onClose && (
				<button
					type='button'
					className='modal__close'
					onClick={onClose}
					disabled={closeDisabled}
					aria-label='Close'
				>
					×
				</button>
			)}
		</div>
	);
};
