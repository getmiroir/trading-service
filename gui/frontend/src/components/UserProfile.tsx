import { type ReactNode, useState } from 'react';

import {
	AccountCheckingNotice,
	AccountLoginPrompt,
	AccountServerOfflineNotice,
} from './AccountAuthBlocks';
import './UserProfile.css';
import { UserProfileButton } from './UserProfileButton';
import { UserProfileModal } from './UserProfileModal';

type UserProfileProps = {
	userStatus?: VerifyResult;
	loginBusy: boolean;
	logoutBusy: boolean;
	manualLoginURL?: string;
	onLogin: () => void;
	onCancelLogin: () => void;
	onLogout: () => void;
	serviceInstallVisible: boolean;
	serviceInstallDisabled: boolean;
	serviceUninstallVisible: boolean;
	serviceUninstallDisabled: boolean;
	serviceUninstallShowBlockedTooltip: boolean;
	onServiceInstall: () => void;
	onServiceUninstall: () => void;
	children?: ReactNode;
};

export const UserProfile = ({
	userStatus,
	loginBusy,
	logoutBusy,
	manualLoginURL,
	onLogin,
	onCancelLogin,
	onLogout,
	serviceInstallVisible,
	serviceInstallDisabled,
	serviceUninstallVisible,
	serviceUninstallDisabled,
	serviceUninstallShowBlockedTooltip,
	onServiceInstall,
	onServiceUninstall,
	children,
}: UserProfileProps) => {
	const [isOpen, setIsOpen] = useState(false);

	const handleLogout = () => {
		setIsOpen(false);
		onLogout();
	};

	if (!userStatus) {
		return <AccountCheckingNotice variant='stacked' />;
	}

	if (userStatus.status === 'unreachable') {
		return <AccountServerOfflineNotice variant='stacked' />;
	}

	if (userStatus.status !== 'valid') {
		return (
			<AccountLoginPrompt
				variant='stacked'
				credentialState={userStatus.status === 'missing' ? 'missing' : 'invalid'}
				loginBusy={loginBusy}
				manualLoginURL={manualLoginURL}
				onLogin={onLogin}
				onCancelLogin={onCancelLogin}
			/>
		);
	}

	const { email, userId, apiKey } = userStatus.auth ?? {};

	return (
		<>
			<UserProfileButton onClick={() => setIsOpen(true)} />

			{children}

			{isOpen && (
				<UserProfileModal
					email={email}
					userId={userId}
					apiKey={apiKey}
					serviceInstallVisible={serviceInstallVisible}
					serviceInstallDisabled={serviceInstallDisabled}
					serviceUninstallVisible={serviceUninstallVisible}
					serviceUninstallDisabled={serviceUninstallDisabled}
					serviceUninstallShowBlockedTooltip={serviceUninstallShowBlockedTooltip}
					onServiceInstall={onServiceInstall}
					onServiceUninstall={onServiceUninstall}
					logoutBusy={logoutBusy}
					onLogout={handleLogout}
					onClose={() => setIsOpen(false)}
				/>
			)}
		</>
	);
};
