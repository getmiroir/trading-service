interface ReportModalTextAreaProps {
	message: string;
	onChange: (value: string) => void;
	disabled: boolean;
}

export const ReportModalTextArea = ({ message, onChange, disabled }: ReportModalTextAreaProps) => (
	<>
		<label htmlFor='report-message' className='modal__label'>
			Describe the issue
		</label>
		<textarea
			id='report-message'
			className='report-textarea'
			placeholder='Please describe what happened…'
			value={message}
			onChange={(e) => onChange(e.target.value)}
			disabled={disabled}
		/>
	</>
);
