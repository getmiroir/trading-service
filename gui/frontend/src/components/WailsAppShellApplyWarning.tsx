import { OneCircleIcon } from './icons/OneCircleIcon';
import { TwoCircleIcon } from './icons/TwoCircleIcon';

type WailsAppShellApplyWarningProps = {
	secondOfTwo?: boolean;
};

export const WailsAppShellApplyWarning = ({
	secondOfTwo = false,
}: WailsAppShellApplyWarningProps) => (
	<div className='install-step-card'>
		<div className='install-step-card__icon'>
			{secondOfTwo ? (
				<TwoCircleIcon width={22} height={22} />
			) : (
				<OneCircleIcon width={22} height={22} />
			)}
		</div>
		<div className='install-step-card__body'>
			<span className='install-step-card__title'>Service will restart</span>
			<span className='install-step-card__desc'>
				Applying these changes will restart the service so it can pick up your updated
				configuration.
			</span>
		</div>
	</div>
);
