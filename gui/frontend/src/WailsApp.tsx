import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { AppBrand } from './components/AppBrand';
import { AppUpdateModal } from './components/AppUpdateModal';
import { ServiceStopConfirmModal } from './components/ServiceStopConfirmModal';
import { UserProfile } from './components/UserProfile';
import { WailsAppShell } from './components/WailsAppShell';
import { useAccountForm } from './hooks/useAccountForm';
import { useAccountLabels } from './hooks/useAccountLabels';
import { useAppUpdate } from './hooks/useAppUpdate';
import { useAuth } from './hooks/useAuth';
import { useHyperliquidAccountForm } from './hooks/useHyperliquidAccountForm';
import { useServiceStatus } from './hooks/useServiceStatus';
import { useWailsEvent } from './hooks/useWailsEvent';
import { isDevelopVersion } from './lib/version';
import { managerApi } from './lib/wails';
import { type ExchangeType } from './types';

const DEEP_LINK_EVENT_NAME = 'deepLink';
const ADD_ACCOUNT_DEEP_LINK = 'miroir://accounts/new';

const isAddAccountDeepLink = (url: string) => {
	const normalised = url.replace(/\/+$/, '').toLowerCase();
	return normalised === ADD_ACCOUNT_DEEP_LINK;
};

export const WailsApp = () => {
	const [pendingCount, setPendingCount] = useState(0);
	const [applySubmitting, setApplySubmitting] = useState(false);
	const [showReport, setShowReport] = useState(false);
	const [loginURL, setLoginURL] = useState<string | undefined>();

	const service = useServiceStatus();
	const appUpdate = useAppUpdate();
	const isDevelopMode = useMemo(() => isDevelopVersion(service.version), [service.version]);

	const markAdded = useCallback(() => {
		if (service.isRunning) {
			setPendingCount((c) => c + 1);
		}
	}, [service.isRunning]);

	const markRemoved = useCallback(() => {
		if (service.isRunning) {
			setPendingCount((c) => Math.max(0, c - 1));
		}
	}, [service.isRunning]);

	const auth = useAuth(markAdded);
	const accountLabels = useAccountLabels(auth.userStatus?.status === 'valid');

	const loadAccountsRef = useRef<() => Promise<void>>(async () => {});
	const hlAccountForm = useHyperliquidAccountForm({
		onAdded: useCallback(() => {
			void loadAccountsRef.current();
			markAdded();
		}, [markAdded]),
	});
	const accountForm = useAccountForm({
		onAdded: markAdded,
		onRemoved: markRemoved,
		onHLOpenAdd: hlAccountForm.openAddForm,
		onHLOpenEdit: hlAccountForm.openEditForm,
		enabledClients: service.enabledClients,
	});
	loadAccountsRef.current = accountForm.loadAccounts;
	const { openAddPicker } = accountForm;

	const handlePickExchange = useCallback(
		(exchange: ExchangeType) => {
			accountForm.openAddForm(exchange);
		},
		[accountForm],
	);

	const handleEditAccount = useCallback(
		(id: string, exchange: ExchangeType) => {
			void accountForm.openEditForm(id, exchange);
		},
		[accountForm],
	);

	const handleDeepLink = useCallback(
		(url: string) => {
			if (!url) {
				return;
			}
			if (isAddAccountDeepLink(url)) {
				openAddPicker();
			}
		},
		[openAddPicker],
	);

	useWailsEvent<string>(DEEP_LINK_EVENT_NAME, handleDeepLink);

	useEffect(() => {
		let cancelled = false;

		const loadLoginURL = async () => {
			try {
				const url = await managerApi.getLoginURL();
				if (!cancelled) {
					setLoginURL(url);
				}
			} catch {
				if (!cancelled) {
					setLoginURL(undefined);
				}
			}
		};

		void loadLoginURL();

		return () => {
			cancelled = true;
		};
	}, []);

	const error = service.error || auth.error || accountForm.error;
	const applyDisabled = pendingCount === 0 || applySubmitting;

	const stopConfirmModal = service.stopConfirmVisible && (
		<ServiceStopConfirmModal onConfirm={service.confirmStop} onCancel={service.cancelStop} />
	);

	const handleApply = async () => {
		setApplySubmitting(true);
		try {
			const ok = await service.restartService();
			if (ok) {
				setPendingCount(0);
			}
		} finally {
			setApplySubmitting(false);
		}
	};

	// Login / checking / offline states — keep centered layout
	if (auth.userStatus?.status !== 'valid') {
		return (
			<div className='container-root'>
				<div className='app-title'>
					<AppBrand />
				</div>

				<UserProfile
					userStatus={auth.userStatus}
					loginBusy={auth.loginBusy}
					logoutBusy={auth.logoutBusy}
					manualLoginURL={auth.manualLoginURL}
					onLogin={() => void auth.handleLogin()}
					onCancelLogin={() => void auth.handleCancelLogin()}
					onLogout={() => void auth.handleLogout()}
					serviceInstallVisible={service.installVisible}
					serviceInstallDisabled={service.installDisabled}
					serviceUninstallVisible={service.uninstallVisible}
					serviceUninstallDisabled={service.uninstallDisabled}
					serviceUninstallShowBlockedTooltip={service.uninstallShowBlockedTooltip}
					onServiceInstall={service.handleInstall}
					onServiceUninstall={service.handleUninstall}
				/>

				{stopConfirmModal}
			</div>
		);
	}

	// Authenticated states (Not Installed / Stopped / Running) — shell layout
	return (
		<>
			<WailsAppShell
				statusText={service.statusText}
				serviceStatusClass={service.serviceStatusClass}
				toggleChecked={service.toggleChecked}
				toggleDisabled={service.toggleDisabled}
				installVisible={service.installVisible}
				installDisabled={service.installDisabled}
				onToggleChange={service.handleToggle}
				onInstall={service.handleInstall}
				version={service.version}
				versionInfo={service.versionInfo}
				showDowngradeCard={service.showDowngradeCard}
				onDowngradeUninstall={service.handleUninstall}
				downgradeUninstallDisabled={service.uninstallDisabled}
				userStatus={auth.userStatus}
				loginBusy={auth.loginBusy}
				logoutBusy={auth.logoutBusy}
				manualLoginURL={auth.manualLoginURL}
				onLogin={() => void auth.handleLogin()}
				onCancelLogin={() => void auth.handleCancelLogin()}
				onLogout={() => void auth.handleLogout()}
				serviceInstallVisible={service.installVisible}
				serviceInstallDisabled={service.installDisabled}
				serviceUninstallVisible={service.uninstallVisible}
				serviceUninstallDisabled={service.uninstallDisabled}
				serviceUninstallShowBlockedTooltip={service.uninstallShowBlockedTooltip}
				onServiceInstall={service.handleInstall}
				onServiceUninstall={service.handleUninstall}
				error={error}
				accounts={accountForm.accounts}
				accountLabels={accountLabels.labels}
				onAddAccount={accountForm.openAddPicker}
				onEditAccount={handleEditAccount}
				onRemoveAccount={(id) => void accountForm.handleRemoveAccount(id)}
				onRenameAccount={accountLabels.renameAccount}
				pendingRemoveID={accountForm.pendingRemoveID}
				onConfirmRemove={() => void accountForm.confirmRemove()}
				onCancelRemove={accountForm.cancelRemove}
				removalBlockedAccountID={accountForm.removalBlockedAccountID}
				onDismissAccountRemovalBlocked={accountForm.dismissAccountRemovalBlocked}
				loginURL={loginURL}
				pendingCount={pendingCount}
				applyDisabled={applyDisabled}
				applySubmitting={applySubmitting}
				onApply={() => void handleApply()}
				overlayMode={accountForm.overlayMode}
				enabledClients={service.enabledClients}
				selectedExchange={accountForm.selectedExchange}
				editAccountID={accountForm.editAccountID}
				referralURL={
					accountForm.selectedExchange
						? accountForm.referralURLs[accountForm.selectedExchange]
						: undefined
				}
				formLoading={accountForm.formLoading}
				formError={accountForm.formError}
				apiKey={accountForm.apiKey}
				apiSecret={accountForm.apiSecret}
				minimumEquity={accountForm.minimumEquity}
				demo={accountForm.demo}
				testnet={accountForm.testnet}
				bybitRegion={accountForm.bybitRegion}
				bybitRegionEnabled={accountForm.bybitRegionEnabled}
				isDevelopMode={isDevelopMode}
				formSubmitting={accountForm.formSubmitting}
				onPickExchange={handlePickExchange}
				onCloseOverlay={accountForm.closeOverlay}
				onSetApiKey={accountForm.setApiKey}
				onSetApiSecret={accountForm.setApiSecret}
				onSetMinimumEquity={accountForm.setMinimumEquity}
				onSetDemo={accountForm.setDemo}
				onSetTestnet={accountForm.setTestnet}
				onSetBybitRegion={accountForm.setBybitRegion}
				onSetBybitRegionEnabled={accountForm.setBybitRegionEnabled}
				onReturnToPicker={accountForm.openAddPicker}
				onAcceptAddNotice={accountForm.acceptAddNotice}
				onSubmitOverlay={() => void accountForm.submitAccount()}
				hlOverlay={{
					editAccountID: hlAccountForm.editAccountID,
					formLoading: hlAccountForm.formLoading,
					formError: hlAccountForm.formError,
					privateKey: hlAccountForm.privateKey,
					masterWalletAddress: hlAccountForm.masterWalletAddress,
					subaccountAddress: hlAccountForm.subaccountAddress,
					minimumEquity: hlAccountForm.minimumEquity,
					isTestNet: hlAccountForm.isTestNet,
					formSubmitting: hlAccountForm.formSubmitting,
					onSetPrivateKey: hlAccountForm.setPrivateKey,
					onSetMasterWalletAddress: hlAccountForm.setMasterWalletAddress,
					onSetSubaccountAddress: hlAccountForm.setSubaccountAddress,
					onSetMinimumEquity: hlAccountForm.setMinimumEquity,
					onSetIsTestNet: hlAccountForm.setIsTestNet,
					onSubmit: () =>
						void hlAccountForm.submitAccount().then((ok) => {
							if (ok) accountForm.closeOverlay();
						}),
				}}
				showReport={showReport}
				onOpenReport={() => setShowReport(true)}
				onCloseReport={() => setShowReport(false)}
			/>

			{stopConfirmModal}

			{appUpdate.prompt && (
				<AppUpdateModal
					currentVersion={appUpdate.prompt.currentVersion}
					latestVersion={appUpdate.prompt.latestVersion}
					onDownload={appUpdate.download}
					onDismiss={appUpdate.dismiss}
				/>
			)}
		</>
	);
};
