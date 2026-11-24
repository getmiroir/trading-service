import { useCallback, useEffect, useState } from 'react';

import { appApi, openExternal } from '../lib/wails';

const UPDATE_CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000;

export function useAppUpdate() {
	const [info, setInfo] = useState<AppUpdateInfo | undefined>();
	const [dismissedVersion, setDismissedVersion] = useState<string | undefined>();

	useEffect(() => {
		let cancelled = false;

		const check = async () => {
			try {
				const result = await appApi.checkAppUpdate();
				if (!cancelled) {
					setInfo(result);
				}
			} catch {
				// An unreachable manifest must never surface in the UI.
			}
		};

		void check();
		const timer = setInterval(() => void check(), UPDATE_CHECK_INTERVAL_MS);

		return () => {
			cancelled = true;
			clearInterval(timer);
		};
	}, []);

	// Dismissing hides the prompt for this version only, so a release published
	// while the app stays open brings it back.
	const dismiss = useCallback(() => {
		setDismissedVersion(info?.latestVersion);
	}, [info]);

	const download = useCallback(() => {
		if (info) {
			void openExternal(info.downloadUrl);
		}
		dismiss();
	}, [info, dismiss]);

	// Undefined while there is nothing to prompt about, so the caller narrows to a
	// fully populated AppUpdateInfo before rendering the modal.
	const prompt =
		info?.updateAvailable === true && info.latestVersion !== dismissedVersion
			? info
			: undefined;

	return { prompt, dismiss, download };
}
