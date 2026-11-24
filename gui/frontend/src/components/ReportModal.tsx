import { useEffect, useState } from 'react';

import { managerApi } from '../lib/wails';
import { Modal } from './Modal';
import { type ModalPhase, type ReportResult } from './ReportModal.utils';
import { ReportModalCheckbox } from './ReportModalCheckbox';
import { ReportModalFormButtons } from './ReportModalFormButtons';
import { ReportModalResult } from './ReportModalResult';
import { ReportModalTextArea } from './ReportModalTextArea';

const REPORT_MAX_LOG_FILES = 2;

interface ReportModalProps {
	isOpen: boolean;
	onClose: () => void;
}

export const ReportModal = ({ isOpen, onClose }: ReportModalProps) => {
	const [phase, setPhase] = useState<ModalPhase>('form');
	const [message, setMessage] = useState('');
	const [attachLogs, setAttachLogs] = useState(false);
	const [logsBundleSize, setLogsBundleSize] = useState(0);
	const [logLoading, setLogLoading] = useState(false);
	const [result, setResult] = useState<ReportResult | undefined>();

	// Fetch log sizes when attachLogs is true (loading/clear handled in checkbox onChange).
	useEffect(() => {
		if (!attachLogs) return;
		let cancelled = false;
		managerApi
			.prepareSupportLogs(REPORT_MAX_LOG_FILES)
			.then((bundle: { serviceLogsSize: number; guiLogsSize: number }) => {
				if (!cancelled) {
					setLogsBundleSize(bundle.serviceLogsSize + bundle.guiLogsSize);
				}
			})
			.catch(() => {
				if (!cancelled) setLogsBundleSize(0);
			})
			.finally(() => {
				if (!cancelled) setLogLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, [attachLogs]);

	const resetState = () => {
		setPhase('form');
		setMessage('');
		setAttachLogs(false);
		setLogsBundleSize(0);
		setLogLoading(false);
		setResult(undefined);
	};

	const handleClose = () => {
		resetState();
		onClose();
	};

	const handleSubmit = async () => {
		if (!message.trim()) return;
		setPhase('submitting');
		try {
			const res = await managerApi.submitReport(
				message.trim(),
				attachLogs,
				REPORT_MAX_LOG_FILES,
			);
			setResult(res);
		} catch {
			setResult({
				success: false,
				message:
					'Your report could not be submitted. Please check your connection and try again.',
			});
		}
		setPhase('done');
	};

	const handleToggleLogs = (checked: boolean) => {
		setAttachLogs(checked);
		if (checked) {
			setLogLoading(true);
			setLogsBundleSize(0);
		} else {
			setLogsBundleSize(0);
			setLogLoading(false);
		}
	};

	if (!isOpen) return null;

	if (phase === 'done' && result) {
		return <ReportModalResult result={result} onClose={handleClose} />;
	}

	const submitting = phase === 'submitting';

	return (
		<Modal
			size='md'
			title='Report an issue'
			onClose={handleClose}
			closeOnEscape={!submitting}
			closeOnBackdropClick={false}
			closeDisabled={submitting}
		>
			<ReportModalTextArea message={message} onChange={setMessage} disabled={submitting} />
			<ReportModalCheckbox
				attachLogs={attachLogs}
				onToggle={handleToggleLogs}
				logLoading={logLoading}
				logsBundleSize={logsBundleSize}
				disabled={submitting}
			/>
			<ReportModalFormButtons
				phase={phase}
				onCancel={handleClose}
				onSubmit={() => void handleSubmit()}
				canSubmit={!!message.trim()}
			/>
		</Modal>
	);
};
