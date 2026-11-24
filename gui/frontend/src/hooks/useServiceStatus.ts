import { useCallback, useEffect, useMemo, useState } from 'react';

import { appApi } from '../lib/wails';
import { Exchange, ExchangeType } from '../types';
import { useWailsEvent } from './useWailsEvent';

type SettledPhase = 'connecting' | 'running' | 'stopped' | 'notinstalled' | 'unknown' | 'failed';

function resolvePhase(raw: string): SettledPhase {
	const normalizedStatus = raw.toLowerCase().replace(/\s+/g, '');
	if (normalizedStatus === 'running') return 'running';
	if (normalizedStatus === 'stopped') return 'stopped';
	if (normalizedStatus === 'notinstalled') return 'notinstalled';
	if (normalizedStatus === 'failed') return 'failed';
	return 'unknown';
}

export function useServiceStatus() {
	const [settledPhase, setSettledPhase] = useState<SettledPhase>('connecting');
	const [settledText, setSettledText] = useState('Connecting...');
	const [transition, setTransition] = useState<{
		text: string;
		toggleChecked: boolean;
	} | null>(null);
	const [error, setError] = useState('');
	const [version, setVersion] = useState('');
	const [enabledClients, setEnabledClients] = useState<ExchangeType[]>([
		Exchange.Bybit,
		Exchange.Binance,
	]);
	const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null);
	const [downgradeDismissed, setDowngradeDismissed] = useState(false);

	const settle = useCallback((status: string) => {
		setSettledPhase(resolvePhase(status));
		setSettledText(status);
		setTransition(null);
	}, []);

	const startTransition = useCallback((text: string, toggleChecked: boolean) => {
		setTransition({ text, toggleChecked });
	}, []);

	const onStatus = useCallback((info: ServiceStatusInfo) => settle(info.status), [settle]);
	const onTransition = useCallback(
		(info: ServiceStatusInfo) => {
			const status = info.status.toLowerCase();
			startTransition(
				info.status,
				status.startsWith('starting') || status.startsWith('restarting'),
			);
		},
		[startTransition],
	);

	useWailsEvent('serviceStatus', onStatus);
	useWailsEvent('serviceTransition', onTransition);

	const applyVersionInfo = useCallback((info: VersionInfo) => {
		setVersionInfo(info);
		if (info.status !== 'downgrade') {
			setDowngradeDismissed(false);
		}
	}, []);

	const onVersionDowngrade = useCallback((info: VersionInfo) => {
		setVersionInfo(info);
		setDowngradeDismissed(false);
	}, []);
	useWailsEvent<VersionInfo>('versionDowngrade', onVersionDowngrade);

	useEffect(() => {
		appApi
			.getStatus()
			.then((info) => settle(info.status))
			.catch((err) => setError(String(err)));
		appApi
			.getVersion()
			.then(setVersion)
			.catch((err) => setError(String(err)));
		appApi
			.getEnabledClients()
			.then((clients) => setEnabledClients(clients as ExchangeType[]))
			.catch((err) => setError(String(err)));
		appApi
			.getVersionStatus()
			.then(applyVersionInfo)
			.catch((err) => setError(String(err)));
	}, [settle, applyVersionInfo]);

	const serviceUI = useMemo(() => {
		const transitioning = transition !== null;
		return {
			statusText: transition?.text ?? settledText,
			serviceStatusClass:
				transitioning || settledPhase === 'connecting' ? 'unknown' : settledPhase,
			toggleChecked: transitioning ? transition.toggleChecked : settledPhase === 'running',
			toggleDisabled:
				transitioning ||
				(settledPhase !== 'running' &&
					settledPhase !== 'stopped' &&
					settledPhase !== 'failed'),
			installVisible: settledPhase === 'notinstalled',
			installDisabled: transitioning,
			uninstallVisible: settledPhase !== 'notinstalled' && settledPhase !== 'connecting',
			uninstallDisabled: transitioning || settledPhase !== 'stopped',
			uninstallShowBlockedTooltip: settledPhase === 'running',
		};
	}, [settledPhase, settledText, transition]);

	const isRunning = settledPhase === 'running';

	const runAction = useCallback(async (fn: () => Promise<void>): Promise<boolean> => {
		setError('');
		try {
			await fn();
			return true;
		} catch (err) {
			setError(String(err));
			return false;
		}
	}, []);

	const [stopConfirmVisible, setStopConfirmVisible] = useState(false);

	const requestStop = useCallback(() => setStopConfirmVisible(true), []);
	useWailsEvent('serviceStopConfirmRequested', requestStop);

	const handleToggle = useCallback(
		(checked: boolean) => {
			if (!checked) {
				requestStop();
				return;
			}
			startTransition('Starting...', true);
			void runAction(() => appApi.startService());
		},
		[requestStop, startTransition, runAction],
	);

	const confirmStop = useCallback(() => {
		setStopConfirmVisible(false);
		startTransition('Stopping...', false);
		void runAction(() => appApi.stopService());
	}, [startTransition, runAction]);

	const cancelStop = useCallback(() => setStopConfirmVisible(false), []);

	const refreshAfterControl = useCallback(() => {
		setDowngradeDismissed(true);
		void appApi
			.getVersionStatus()
			.then(applyVersionInfo)
			.catch(() => {});
	}, [applyVersionInfo]);

	const handleInstall = useCallback(() => {
		startTransition('Installing...', false);
		void (async () => {
			if (await runAction(() => appApi.installService())) {
				refreshAfterControl();
			}
		})();
	}, [startTransition, runAction, refreshAfterControl]);

	const handleUninstall = useCallback(() => {
		startTransition('Uninstalling...', false);
		void (async () => {
			if (await runAction(() => appApi.uninstallService())) {
				refreshAfterControl();
			}
		})();
	}, [startTransition, runAction, refreshAfterControl]);

	const restartService = useCallback(() => runAction(() => appApi.restartService()), [runAction]);

	const showDowngradeCard = versionInfo?.status === 'downgrade' && !downgradeDismissed;

	return {
		...serviceUI,
		isRunning,
		error,
		version,
		enabledClients,
		versionInfo,
		showDowngradeCard,
		clearError: useCallback(() => setError(''), []),
		handleToggle,
		stopConfirmVisible,
		confirmStop,
		cancelStop,
		handleInstall,
		handleUninstall,
		restartService,
	};
}
