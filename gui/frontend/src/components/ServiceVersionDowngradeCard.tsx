import { VersionComparison } from './VersionComparison';

type ServiceVersionDowngradeCardProps = {
	guiVersion: string;
	serviceVersion: string;
	disabled: boolean;
	onUninstall: () => void;
};

export const ServiceVersionDowngradeCard = ({
	guiVersion,
	serviceVersion,
	disabled,
	onUninstall,
}: ServiceVersionDowngradeCardProps) => (
	<>
		<div className='install-step-card install-step-card--warning'>
			<div className='install-step-card__body'>
				<span className='install-step-card__title'>Version mismatch detected</span>
				<span className='install-step-card__desc'>
					The installed service is newer than this application. Uninstalling will replace
					it with the version bundled here. Pause the service to continue the downgrade
					flow.
				</span>
				<VersionComparison
					rows={[
						{
							label: 'Currently installed',
							value: serviceVersion,
							variant: 'outgoing',
						},
						{ label: 'Will be installed', value: guiVersion, variant: 'incoming' },
					]}
				/>
			</div>
		</div>
		<button
			type='button'
			className='apply-btn install-step-card__btn'
			disabled={disabled}
			onClick={onUninstall}
		>
			Uninstall {serviceVersion}
		</button>
	</>
);
