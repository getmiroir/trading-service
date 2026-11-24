import { useCallback, useEffect, useState } from 'react';

import { managerApi } from '../lib/wails';

const LABELS_REFRESH_INTERVAL_MS = 60_000;

// Account labels live on the backend (control API); fetching is best-effort so
// the GUI falls back to raw account IDs when offline or logged out.
export const useAccountLabels = (enabled: boolean) => {
	const [labels, setLabels] = useState<Record<string, string>>({});

	const loadLabels = useCallback(async () => {
		try {
			const next = await managerApi.getAccountLabels();
			setLabels(next ?? {});
		} catch {
			// Offline or logged out — keep showing raw account IDs.
		}
	}, []);

	useEffect(() => {
		if (!enabled) {
			return;
		}
		let cancelled = false;

		const load = async () => {
			try {
				const next = await managerApi.getAccountLabels();
				if (!cancelled) {
					setLabels(next ?? {});
				}
			} catch {
				// Offline or logged out — keep showing raw account IDs.
			}
		};

		void load();
		const interval = setInterval(() => void load(), LABELS_REFRESH_INTERVAL_MS);

		return () => {
			cancelled = true;
			clearInterval(interval);
		};
	}, [enabled]);

	const renameAccount = useCallback(
		async (accountID: string, label: string) => {
			await managerApi.setAccountLabel(accountID, label.trim());
			await loadLabels();
		},
		[loadLabels],
	);

	return { labels, loadLabels, renameAccount };
};
