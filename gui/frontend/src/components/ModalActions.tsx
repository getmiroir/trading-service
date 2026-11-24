import { type ReactNode } from 'react';

type ModalActionsProps = {
	children: ReactNode;
};

export const ModalActions = ({ children }: ModalActionsProps) => (
	<div className='modal__actions'>{children}</div>
);
