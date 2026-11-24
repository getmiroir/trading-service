import { PowerIcon } from './icons/PowerIcon';

type ServiceStatusPowerButtonProps = {
	checked: boolean;
	disabled: boolean;
	statusText: string;
	serviceStatusClass: string;
	onToggle: (checked: boolean) => void;
};

export const ServiceStatusPowerButton = ({
	checked,
	disabled,
	statusText,
	serviceStatusClass,
	onToggle,
}: ServiceStatusPowerButtonProps) => (
	<div className='power-btn-wrap'>
		<div className={`power-btn-ring power-btn-ring--${serviceStatusClass}`}>
			<button
				type='button'
				className={`power-btn power-btn--${serviceStatusClass}`}
				disabled={disabled}
				onClick={() => onToggle(!checked)}
				aria-label={checked ? 'Stop service' : 'Start service'}
				aria-pressed={checked}
			>
				<PowerIcon width={64} height={64} />
			</button>
		</div>
		<div className='power-status-label'>
			<span className={`dot ${serviceStatusClass}`} />
			<span>{statusText}</span>
		</div>
	</div>
);
