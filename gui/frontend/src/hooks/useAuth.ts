import { useCallback, useEffect, useState } from 'react';

import { appApi, managerApi } from '../lib/wails';
import { useWailsEvent } from './useWailsEvent';

export function useAuth(onChanged?: () => void) {
	const [userStatus, setUserStatus] = useState<VerifyResult | undefined>();
	const [loginBusy, setLoginBusy] = useState(false);
	const [logoutBusy, setLogoutBusy] = useState(false);
	const [manualLoginURL, setManualLoginURL] = useState<string | undefined>();
	const [error, setError] = useState('');

	// The backend emits the fully-built login URL (callback port + PKCE) once it
	// opens the browser, so we can offer a manual open/copy fallback.
	useWailsEvent<string>('login:url', setManualLoginURL);

	const refresh = useCallback(async () => {
		try {
			const result = await managerApi.verifyControlApiKey();
			setUserStatus(result);
		} catch {
			setUserStatus({ status: 'unreachable' });
		}
	}, []);

	useEffect(() => {
		void refresh();
	}, [refresh]);

	const handleLogin = useCallback(async () => {
		setLoginBusy(true);
		setManualLoginURL(undefined);
		setError('');
		try {
			await appApi.login();
			await refresh();
			onChanged?.();
		} catch (err) {
			setError(String(err));
		} finally {
			setLoginBusy(false);
			setManualLoginURL(undefined);
		}
	}, [refresh, onChanged]);

	const handleCancelLogin = useCallback(async () => {
		// Unblocks the pending appApi.login() call, which then settles handleLogin.
		await appApi.cancelLogin();
	}, []);

	const handleLogout = useCallback(async () => {
		setLogoutBusy(true);
		setError('');
		try {
			await managerApi.deleteControlApiKey();
			await refresh();
			onChanged?.();
		} catch (err) {
			setError(String(err));
		} finally {
			setLogoutBusy(false);
		}
	}, [refresh, onChanged]);

	return {
		userStatus,
		loginBusy,
		logoutBusy,
		manualLoginURL,
		error,
		handleLogin,
		handleCancelLogin,
		handleLogout,
	};
}
