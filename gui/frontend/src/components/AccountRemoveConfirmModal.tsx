import { ConfirmModal } from './ConfirmModal';

const REMOVE_ACCOUNT_DETAILS = [
	"Its API credentials are deleted from this device. You'll need to enter them again to add it back.",
	'The change takes effect once you apply pending changes, which restarts the service.',
];

type AccountRemoveConfirmModalProps = {
	accountName: string;
	onConfirm: () => void;
	onCancel: () => void;
};

export const AccountRemoveConfirmModal = ({
	accountName,
	onConfirm,
	onCancel,
}: AccountRemoveConfirmModalProps) => (
	<ConfirmModal
		title={`Remove "${accountName}"?`}
		message='Miroir will stop using this account on this device.'
		details={REMOVE_ACCOUNT_DETAILS}
		confirmLabel='Remove account'
		onConfirm={onConfirm}
		onCancel={onCancel}
	/>
);
