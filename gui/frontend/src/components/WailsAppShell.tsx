import { useLayoutEffect, useRef } from 'react';

import { getAccountDisplayName } from '../lib/accountDisplayName';
import { type BybitRegion, type ExchangeType, type OverlayMode } from '../types';
import { AccountOverlay, type HyperliquidOverlayState } from './AccountOverlay';
import { AccountRemovalBlockedModal } from './AccountRemovalBlockedModal';
import { AccountRemoveConfirmModal } from './AccountRemoveConfirmModal';
import { AccountsSection } from './AccountsSection';
import { AppBrand } from './AppBrand';
import { ReportModal } from './ReportModal';
import { ServiceStatus } from './ServiceStatus';
import { ServiceStatusInstallCard } from './ServiceStatusInstallCard';
import { ServiceVersionDowngradeCard } from './ServiceVersionDowngradeCard';
import { UserProfile } from './UserProfile';
import { WailsAppBottomBar } from './WailsAppBottomBar';
import { WailsAppShellApplyButton } from './WailsAppShellApplyButton';
import { WailsAppShellApplyWarning } from './WailsAppShellApplyWarning';
import { WailsAppShellNotifications } from './WailsAppShellNotifications';

type WailsAppShellProps = {
	// Service status
	statusText: string;
	serviceStatusClass: string;
	toggleChecked: boolean;
	toggleDisabled: boolean;
	installVisible: boolean;
	installDisabled: boolean;
	onToggleChange: (checked: boolean) => void;
	onInstall: () => void;
	version: string;
	versionInfo?: VersionInfo | null;
	showDowngradeCard: boolean;
	onDowngradeUninstall: () => void;
	downgradeUninstallDisabled: boolean;

	// Auth / profile
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

	// Error
	error: string;

	// Accounts
	accounts: AccountInfo[];
	accountLabels: Record<string, string>;
	onAddAccount: () => void;
	onEditAccount: (id: string, exchange: ExchangeType) => void;
	onRemoveAccount: (id: string) => void;
	onRenameAccount: (id: string, label: string) => Promise<void>;

	// Confirm remove modal
	pendingRemoveID?: string;
	onConfirmRemove: () => void;
	onCancelRemove: () => void;

	// Mirroring blocked modal
	removalBlockedAccountID?: string;
	onDismissAccountRemovalBlocked: () => void;
	loginURL?: string;

	// Apply button
	pendingCount: number;
	applyDisabled: boolean;
	applySubmitting: boolean;
	onApply: () => void;

	// Account overlay
	overlayMode: OverlayMode;
	enabledClients: ExchangeType[];
	selectedExchange?: ExchangeType;
	editAccountID?: string;
	referralURL?: string;
	formLoading: boolean;
	formError: string;
	apiKey: string;
	apiSecret: string;
	minimumEquity: string;
	demo: boolean;
	testnet: boolean;
	bybitRegion: BybitRegion;
	bybitRegionEnabled: boolean;
	isDevelopMode: boolean;
	formSubmitting: boolean;
	onPickExchange: (exchange: ExchangeType) => void;
	onCloseOverlay: () => void;
	onSetApiKey: (value: string) => void;
	onSetApiSecret: (value: string) => void;
	onSetMinimumEquity: (value: string) => void;
	onSetDemo: (value: boolean) => void;
	onSetTestnet: (value: boolean) => void;
	onSetBybitRegion: (value: BybitRegion) => void;
	onSetBybitRegionEnabled: (value: boolean) => void;
	onReturnToPicker: () => void;
	onAcceptAddNotice: () => void;
	onSubmitOverlay: () => void;
	hlOverlay: HyperliquidOverlayState;

	// Report modal
	showReport: boolean;
	onOpenReport: () => void;
	onCloseReport: () => void;
};

export const WailsAppShell = ({
	statusText,
	serviceStatusClass,
	toggleChecked,
	toggleDisabled,
	installVisible,
	installDisabled,
	onToggleChange,
	onInstall,
	version,
	versionInfo,
	showDowngradeCard,
	onDowngradeUninstall,
	downgradeUninstallDisabled,
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
	error,
	accounts,
	accountLabels,
	onAddAccount,
	onEditAccount,
	onRemoveAccount,
	onRenameAccount,
	pendingRemoveID,
	onConfirmRemove,
	onCancelRemove,
	removalBlockedAccountID,
	onDismissAccountRemovalBlocked,
	loginURL,
	pendingCount,
	applyDisabled,
	applySubmitting,
	onApply,
	overlayMode,
	enabledClients,
	selectedExchange,
	editAccountID,
	referralURL,
	formLoading,
	formError,
	apiKey,
	apiSecret,
	minimumEquity,
	demo,
	testnet,
	bybitRegion,
	bybitRegionEnabled,
	isDevelopMode,
	formSubmitting,
	onPickExchange,
	onCloseOverlay,
	onSetApiKey,
	onSetApiSecret,
	onSetMinimumEquity,
	onSetDemo,
	onSetTestnet,
	onSetBybitRegion,
	onSetBybitRegionEnabled,
	onReturnToPicker,
	onAcceptAddNotice,
	onSubmitOverlay,
	hlOverlay,
	showReport,
	onOpenReport,
	onCloseReport,
}: WailsAppShellProps) => {
	const notificationsScrollRef = useRef<HTMLDivElement>(null);

	useLayoutEffect(() => {
		const el = notificationsScrollRef.current;
		if (el) {
			el.scrollTop = 0;
		}
	}, [installVisible, pendingCount, showDowngradeCard]);

	return (
		<div className='app-shell'>
			<div className='app-shell__topbar'>
				<div className='app-shell__topbar-leading' aria-hidden={true} />
				<AppBrand />
				<div className='app-shell__topbar-trailing' aria-hidden={true} />
			</div>

			<div className='app-shell__center'>
				<ServiceStatus
					statusText={statusText}
					serviceStatusClass={serviceStatusClass}
					toggleChecked={toggleChecked}
					toggleDisabled={toggleDisabled}
					installVisible={installVisible}
					onToggleChange={onToggleChange}
				/>
				<div className='error'>{error}</div>
			</div>

			{(installVisible || pendingCount > 0 || showDowngradeCard) && (
				<div className='app-shell__shell-cards'>
					<WailsAppShellNotifications ref={notificationsScrollRef}>
						{showDowngradeCard && versionInfo && (
							<div className='app-shell-notifications__group'>
								<ServiceVersionDowngradeCard
									guiVersion={versionInfo.guiVersion}
									serviceVersion={versionInfo.serviceVersion}
									disabled={downgradeUninstallDisabled}
									onUninstall={onDowngradeUninstall}
								/>
							</div>
						)}
						{installVisible && (
							<div className='app-shell-notifications__group'>
								<ServiceStatusInstallCard
									disabled={installDisabled}
									onInstall={onInstall}
								/>
							</div>
						)}
						{pendingCount > 0 && (
							<div className='app-shell-notifications__group'>
								<WailsAppShellApplyWarning
									secondOfTwo={installVisible && pendingCount > 0}
								/>
								<WailsAppShellApplyButton
									pendingCount={pendingCount}
									disabled={applyDisabled}
									submitting={applySubmitting}
									onApply={onApply}
								/>
							</div>
						)}
					</WailsAppShellNotifications>
				</div>
			)}

			<div className='app-shell__accounts'>
				<div className={installVisible ? 'accounts-ghost' : ''}>
					<AccountsSection
						accounts={accounts}
						accountLabels={accountLabels}
						onAdd={onAddAccount}
						onEdit={onEditAccount}
						onRemove={onRemoveAccount}
						onRename={onRenameAccount}
					/>
				</div>
			</div>

			<WailsAppBottomBar version={version} onReport={onOpenReport} />

			<UserProfile
				userStatus={userStatus}
				loginBusy={loginBusy}
				logoutBusy={logoutBusy}
				manualLoginURL={manualLoginURL}
				onLogin={onLogin}
				onCancelLogin={onCancelLogin}
				onLogout={onLogout}
				serviceInstallVisible={serviceInstallVisible}
				serviceInstallDisabled={serviceInstallDisabled}
				serviceUninstallVisible={serviceUninstallVisible}
				serviceUninstallDisabled={serviceUninstallDisabled}
				serviceUninstallShowBlockedTooltip={serviceUninstallShowBlockedTooltip}
				onServiceInstall={onServiceInstall}
				onServiceUninstall={onServiceUninstall}
			/>

			{pendingRemoveID && (
				<AccountRemoveConfirmModal
					accountName={getAccountDisplayName(pendingRemoveID, accountLabels)}
					onConfirm={onConfirmRemove}
					onCancel={onCancelRemove}
				/>
			)}

			{removalBlockedAccountID && (
				<AccountRemovalBlockedModal
					accountID={removalBlockedAccountID}
					accountName={getAccountDisplayName(removalBlockedAccountID, accountLabels)}
					loginURL={loginURL}
					onDismiss={onDismissAccountRemovalBlocked}
				/>
			)}

			<AccountOverlay
				overlayMode={overlayMode}
				enabledClients={enabledClients}
				selectedExchange={selectedExchange}
				editAccountID={editAccountID}
				editAccountName={
					editAccountID ? getAccountDisplayName(editAccountID, accountLabels) : undefined
				}
				referralURL={referralURL}
				formLoading={formLoading}
				formError={formError}
				apiKey={apiKey}
				apiSecret={apiSecret}
				minimumEquity={minimumEquity}
				demo={demo}
				testnet={testnet}
				bybitRegion={bybitRegion}
				bybitRegionEnabled={bybitRegionEnabled}
				isDevelopMode={isDevelopMode}
				formSubmitting={formSubmitting}
				onPickExchange={onPickExchange}
				onClose={onCloseOverlay}
				onSetApiKey={onSetApiKey}
				onSetApiSecret={onSetApiSecret}
				onSetMinimumEquity={onSetMinimumEquity}
				onSetDemo={onSetDemo}
				onSetTestnet={onSetTestnet}
				onSetBybitRegion={onSetBybitRegion}
				onSetBybitRegionEnabled={onSetBybitRegionEnabled}
				onReturnToPicker={onReturnToPicker}
				onAcceptAddNotice={onAcceptAddNotice}
				onSubmit={onSubmitOverlay}
				hlOverlay={hlOverlay}
			/>

			<ReportModal isOpen={showReport} onClose={onCloseReport} />
		</div>
	);
};
