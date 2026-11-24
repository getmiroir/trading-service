import { OneCircleIcon } from './icons/OneCircleIcon';

type ServiceStatusInstallCardProps = {
	disabled: boolean;
	onInstall: () => void;
};

export const ServiceStatusInstallCard = ({
	disabled,
	onInstall,
}: ServiceStatusInstallCardProps) => (
	<>
		<div className='install-step-card'>
			<div className='install-step-card__icon'>
				<OneCircleIcon width={22} height={22} />
			</div>
			<div className='install-step-card__body'>
				<span className='install-step-card__title'>Service not installed</span>
				<span className='install-step-card__desc'>
					Install the system service to enable account management and start trading
					automation.
				</span>
			</div>
		</div>
		<button
			type='button'
			className='apply-btn install-step-card__btn'
			disabled={disabled}
			onClick={onInstall}
		>
			Install service
		</button>
	</>
);
