import { LockIcon } from './icons/LockIcon';

type ServiceStatusLockButtonProps = {
	statusText: string;
	serviceStatusClass: string;
};

export const ServiceStatusLockButton = ({
	statusText,
	serviceStatusClass,
}: ServiceStatusLockButtonProps) => (
	<div className='power-btn-wrap'>
		<div className='power-btn-ring power-btn-ring--locked'>
			<div className='power-btn power-btn--locked' aria-hidden='true'>
				<LockIcon width={64} height={64} />
			</div>
		</div>
		<div className='power-status-label'>
			<span className={`dot ${serviceStatusClass}`} />
			<span>{statusText}</span>
		</div>
	</div>
);
