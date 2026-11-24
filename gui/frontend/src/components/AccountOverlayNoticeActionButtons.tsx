import { ModalActions } from './ModalActions';
import { ModalButton } from './ModalButton';

type AccountOverlayNoticeActionButtonsProps = {
	acceptDisabled: boolean;
	onReturn: () => void;
	onAccept: () => void;
};

export const AccountOverlayNoticeActionButtons = ({
	acceptDisabled,
	onReturn,
	onAccept,
}: AccountOverlayNoticeActionButtonsProps) => (
	<ModalActions>
		<ModalButton variant='secondary' onClick={onReturn}>
			Return
		</ModalButton>
		<ModalButton
			variant='primary'
			onClick={() => {
				if (acceptDisabled) return;
				onAccept();
			}}
			disabled={acceptDisabled}
		>
			Accept
		</ModalButton>
	</ModalActions>
);
