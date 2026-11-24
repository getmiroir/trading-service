import { formatDate } from '../lib/formatDate';
import { LogoutButton } from './LogoutButton';
import { Modal } from './Modal';
import { ProfileServiceActions } from './ProfileServiceActions';

type ProfileApiKey = {
	keyId: string;
	name: string;
	createdAt: string;
};

type UserProfileModalProps = {
	email?: string;
	userId?: string;
	apiKey?: ProfileApiKey;
	serviceInstallVisible: boolean;
	serviceInstallDisabled: boolean;
	serviceUninstallVisible: boolean;
	serviceUninstallDisabled: boolean;
	serviceUninstallShowBlockedTooltip: boolean;
	onServiceInstall: () => void;
	onServiceUninstall: () => void;
	logoutBusy: boolean;
	onLogout: () => void;
	onClose: () => void;
};

export const UserProfileModal = ({
	email,
	userId,
	apiKey,
	serviceInstallVisible,
	serviceInstallDisabled,
	serviceUninstallVisible,
	serviceUninstallDisabled,
	serviceUninstallShowBlockedTooltip,
	onServiceInstall,
	onServiceUninstall,
	logoutBusy,
	onLogout,
	onClose,
}: UserProfileModalProps) => {
	const displayName = email || userId || 'User';

	return (
		<Modal title='Profile' onClose={onClose}>
			<div className='profile-row'>
				<span className='profile-row__label'>Account</span>
				<span className='profile-row__value'>{displayName}</span>
			</div>

			{apiKey && (
				<div className='profile-api-key'>
					<span className='profile-api-key__label'>API Key</span>
					<span className='profile-api-key__name'>{apiKey.name}</span>
					<span className='profile-api-key__date'>{formatDate(apiKey.createdAt)}</span>
					<div className='profile-api-key__key-id'>
						<span className='profile-api-key__key-id-label'>Key ID</span>
						<code className='profile-api-key__key-id-value'>{apiKey.keyId}</code>
					</div>
				</div>
			)}

			<div className='profile-actions'>
				<ProfileServiceActions
					installVisible={serviceInstallVisible}
					installDisabled={serviceInstallDisabled}
					uninstallVisible={serviceUninstallVisible}
					uninstallDisabled={serviceUninstallDisabled}
					uninstallShowBlockedTooltip={serviceUninstallShowBlockedTooltip}
					onInstall={onServiceInstall}
					onUninstall={onServiceUninstall}
				/>
				<LogoutButton busy={logoutBusy} onLogout={onLogout} />
			</div>
		</Modal>
	);
};
