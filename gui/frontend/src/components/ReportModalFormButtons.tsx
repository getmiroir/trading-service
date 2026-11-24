import { ModalActions } from './ModalActions';
import { ModalButton } from './ModalButton';
import { type ModalPhase } from './ReportModal.utils';

interface ReportModalFormButtonsProps {
	phase: ModalPhase;
	onCancel: () => void;
	onSubmit: () => void;
	canSubmit: boolean;
}

export const ReportModalFormButtons = ({
	phase,
	onCancel,
	onSubmit,
	canSubmit,
}: ReportModalFormButtonsProps) => (
	<ModalActions>
		<ModalButton variant='secondary' onClick={onCancel} disabled={phase === 'submitting'}>
			Cancel
		</ModalButton>
		<ModalButton
			variant='primary'
			onClick={onSubmit}
			disabled={phase === 'submitting' || !canSubmit}
		>
			{phase === 'submitting' ? 'Submitting…' : 'Submit'}
		</ModalButton>
	</ModalActions>
);
