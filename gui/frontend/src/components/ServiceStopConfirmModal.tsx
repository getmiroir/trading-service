import { ConfirmModal } from './ConfirmModal';

const STOP_SERVICE_DETAILS = [
	'Open positions stay open, even if you stop mirroring from the web interface.',
	'To stop trading, stop mirroring on the web interface first or stop the service, then close your positions manually on your exchange.',
];

type ServiceStopConfirmModalProps = {
	onConfirm: () => void;
	onCancel: () => void;
};

export const ServiceStopConfirmModal = ({ onConfirm, onCancel }: ServiceStopConfirmModalProps) => (
	<ConfirmModal
		tone='warning'
		title='Stop the service?'
		message='Miroir will stop updating your positions until you start the service again.'
		details={STOP_SERVICE_DETAILS}
		cancelLabel='Keep running'
		confirmLabel='Stop service'
		onConfirm={onConfirm}
		onCancel={onCancel}
	/>
);
