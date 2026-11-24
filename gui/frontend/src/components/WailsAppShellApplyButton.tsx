type WailsAppShellApplyButtonProps = {
	pendingCount: number;
	disabled: boolean;
	submitting: boolean;
	onApply: () => void;
};

export const WailsAppShellApplyButton = ({
	pendingCount,
	disabled,
	submitting,
	onApply,
}: WailsAppShellApplyButtonProps) => (
	<div className='app-shell__apply'>
		<button
			className={`apply-btn${pendingCount > 0 ? ' apply-btn--pending' : ''}`}
			disabled={disabled}
			onClick={onApply}
		>
			{submitting ? 'Restarting...' : 'Apply changes'}
			{pendingCount > 0 && <span className='apply-badge'>{pendingCount}</span>}
		</button>
	</div>
);
