import {
	AccountCheckingNotice,
	AccountLoginPrompt,
	AccountServerOfflineNotice,
} from './AccountAuthBlocks';

type UserAuthProps = {
	userStatus?: VerifyResult;
	loginBusy: boolean;
	logoutBusy: boolean;
	manualLoginURL?: string;
	onLogin: () => void;
	onCancelLogin: () => void;
	onLogout: () => void;
};

export const UserAuth = ({
	userStatus,
	loginBusy,
	logoutBusy,
	manualLoginURL,
	onLogin,
	onCancelLogin,
	onLogout,
}: UserAuthProps) => {
	if (!userStatus) {
		return <AccountCheckingNotice variant='inline' />;
	}

	if (userStatus.status === 'valid') {
		const email = userStatus.auth?.email || userStatus.auth?.userId || 'User';
		const keyName = userStatus.auth?.apiKey?.name;
		return (
			<>
				<div className='user-info'>
					User
					<span className='user-email'>{email}</span>
					{keyName ? <span className='user-key-name'>Key: {keyName}</span> : null}
				</div>
				<button className='login-btn' disabled={logoutBusy} onClick={onLogout}>
					{logoutBusy ? 'Logging out…' : 'Log out'}
				</button>
			</>
		);
	}

	if (userStatus.status === 'unreachable') {
		return <AccountServerOfflineNotice variant='inline' />;
	}

	return (
		<AccountLoginPrompt
			variant='inline'
			credentialState={userStatus.status === 'missing' ? 'missing' : 'invalid'}
			loginBusy={loginBusy}
			manualLoginURL={manualLoginURL}
			onLogin={onLogin}
			onCancelLogin={onCancelLogin}
		/>
	);
};
