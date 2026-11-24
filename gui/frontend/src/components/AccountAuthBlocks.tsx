import { type ReactNode, useState } from 'react';

import { openExternal } from '../lib/wails';

export const WARNING_SYMBOL = '⚠';

export type AccountAuthVariant = 'stacked' | 'inline';

function withStackedWrap(variant: AccountAuthVariant, inner: ReactNode): ReactNode {
	if (variant === 'stacked') {
		return <div className='profile-login-wrap'>{inner}</div>;
	}
	return inner;
}

type AccountCheckingNoticeProps = {
	variant: AccountAuthVariant;
};

export const AccountCheckingNotice = ({ variant }: AccountCheckingNoticeProps) =>
	withStackedWrap(variant, <span>Checking account…</span>);

type AccountServerOfflineNoticeProps = {
	variant: AccountAuthVariant;
};

export const AccountServerOfflineNotice = ({ variant }: AccountServerOfflineNoticeProps) =>
	withStackedWrap(
		variant,
		<span className='offline-label'>{WARNING_SYMBOL} Server offline</span>,
	);

type AccountLoginPromptProps = {
	variant: AccountAuthVariant;
	credentialState: 'missing' | 'invalid';
	loginBusy: boolean;
	manualLoginURL?: string;
	onLogin: () => void;
	onCancelLogin: () => void;
};

// Shown while a login is in flight: lets the user cancel (so a closed browser
// tab never leaves them stuck) and, if the default browser failed to open,
// open/copy the real login URL manually.
const LoginBusyControls = ({
	manualLoginURL,
	onCancelLogin,
}: {
	manualLoginURL?: string;
	onCancelLogin: () => void;
}) => {
	const [copied, setCopied] = useState(false);

	const handleCopy = () => {
		if (!manualLoginURL) {
			return;
		}
		void navigator.clipboard.writeText(manualLoginURL).then(
			() => setCopied(true),
			() => setCopied(false),
		);
	};

	return (
		<div className='login-busy'>
			<div className='login-busy__status'>
				<button className='login-btn' onClick={onCancelLogin}>
					Cancel
				</button>
			</div>
			{manualLoginURL && (
				<p className='login-manual'>
					Browser didn’t open?{' '}
					<button
						className='login-manual__link'
						onClick={() => openExternal(manualLoginURL)}
					>
						Open the login page
					</button>
					{' · '}
					<button className='login-manual__link' onClick={handleCopy}>
						{copied ? 'Copied!' : 'Copy link'}
					</button>
				</p>
			)}
		</div>
	);
};

export const AccountLoginPrompt = ({
	variant,
	credentialState,
	loginBusy,
	manualLoginURL,
	onLogin,
	onCancelLogin,
}: AccountLoginPromptProps) => {
	const message = loginBusy
		? 'Logging in…'
		: credentialState === 'missing'
			? 'Not logged in'
			: 'Login credentials invalid';
	const content = loginBusy ? (
		<LoginBusyControls manualLoginURL={manualLoginURL} onCancelLogin={onCancelLogin} />
	) : (
		<button className='login-btn btn-cta' onClick={onLogin}>
			Log in
		</button>
	);

	if (variant === 'stacked') {
		return (
			<div className='profile-login-wrap'>
				<p>{message}</p>
				{content}
			</div>
		);
	}

	return (
		<>
			<span>{message}</span>
			{content}
		</>
	);
};
