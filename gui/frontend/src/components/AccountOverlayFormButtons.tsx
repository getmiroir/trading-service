import { ModalActions } from './ModalActions';
import { ModalButton } from './ModalButton';

type AccountOverlayFormButtonsProps = {
	editAccountID?: string;
	formSubmitting: boolean;
	onClose: () => void;
	onSubmit: () => void;
};

export const AccountOverlayFormButtons = ({
	editAccountID,
	formSubmitting,
	onClose,
	onSubmit,
}: AccountOverlayFormButtonsProps) => (
	<ModalActions>
		<ModalButton variant='secondary' onClick={onClose} disabled={formSubmitting}>
			Cancel
		</ModalButton>
		<ModalButton variant='primary' onClick={onSubmit} disabled={formSubmitting}>
			{formSubmitting
				? editAccountID
					? 'Saving…'
					: 'Validating…'
				: editAccountID
					? 'Save changes'
					: 'Add account'}
		</ModalButton>
	</ModalActions>
);
