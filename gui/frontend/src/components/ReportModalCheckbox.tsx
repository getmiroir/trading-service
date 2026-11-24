import { formatBytes } from './ReportModal.utils';

interface ReportModalCheckboxProps {
	attachLogs: boolean;
	onToggle: (checked: boolean) => void;
	logLoading: boolean;
	logsBundleSize: number;
	disabled: boolean;
}

export const ReportModalCheckbox = ({
	attachLogs,
	onToggle,
	logLoading,
	logsBundleSize,
	disabled,
}: ReportModalCheckboxProps) => (
	<div className='report-checkbox-row'>
		<input
			id='report-attach-logs'
			type='checkbox'
			checked={attachLogs}
			onChange={(e) => onToggle(e.target.checked)}
			disabled={disabled}
		/>
		<label htmlFor='report-attach-logs'>Attach last 24 hours of logs</label>
		{attachLogs && (
			<span className='report-log-size'>
				{logLoading ? 'Calculating…' : formatBytes(logsBundleSize)}
			</span>
		)}
	</div>
);
