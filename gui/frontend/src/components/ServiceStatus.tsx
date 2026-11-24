import { ServiceStatusLockButton } from './ServiceStatusLockButton';
import { ServiceStatusPowerButton } from './ServiceStatusPowerButton';

type ServiceStatusProps = {
	statusText: string;
	serviceStatusClass: string;
	toggleChecked: boolean;
	toggleDisabled: boolean;
	installVisible: boolean;
	onToggleChange: (checked: boolean) => void;
};

export const ServiceStatus = ({
	statusText,
	serviceStatusClass,
	toggleChecked,
	toggleDisabled,
	installVisible,
	onToggleChange,
}: ServiceStatusProps) => {
	if (installVisible) {
		return (
			<ServiceStatusLockButton
				statusText='Install to enable'
				serviceStatusClass={serviceStatusClass}
			/>
		);
	}

	return (
		<ServiceStatusPowerButton
			checked={toggleChecked}
			disabled={toggleDisabled}
			statusText={statusText}
			serviceStatusClass={serviceStatusClass}
			onToggle={onToggleChange}
		/>
	);
};
