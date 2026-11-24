import { BybitRegion, Exchange, ExchangeType, OverlayMode } from '../types';
import { AccountOverlayBody } from './AccountOverlayBody';
import { AccountOverlayExchangeCredentials } from './AccountOverlayExchangeCredentials';
import { AccountOverlayExchangeOptions } from './AccountOverlayExchangeOptions';
import { AccountOverlayForm } from './AccountOverlayForm';
import { AccountOverlayHyperliquidCredentials } from './AccountOverlayHyperliquidCredentials';
import { AccountOverlayHyperliquidOptions } from './AccountOverlayHyperliquidOptions';
import { AccountOverlayNotice } from './AccountOverlayNotice';
import { AccountOverlayPicker } from './AccountOverlayPicker';
import { ModalBackdrop } from './ModalBackdrop';

export type HyperliquidOverlayState = {
	editAccountID?: string;
	formLoading: boolean;
	formError: string;
	privateKey: string;
	masterWalletAddress: string;
	subaccountAddress: string;
	minimumEquity: string;
	isTestNet: boolean;
	formSubmitting: boolean;
	onSetPrivateKey: (value: string) => void;
	onSetMasterWalletAddress: (value: string) => void;
	onSetSubaccountAddress: (value: string) => void;
	onSetMinimumEquity: (value: string) => void;
	onSetIsTestNet: (value: boolean) => void;
	onSubmit: () => void;
};

export type AccountOverlayProps = {
	overlayMode: OverlayMode;
	enabledClients: ExchangeType[];
	selectedExchange?: ExchangeType;
	editAccountID?: string;
	editAccountName?: string;
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
	onClose: () => void;
	onSetApiKey: (value: string) => void;
	onSetApiSecret: (value: string) => void;
	onSetMinimumEquity: (value: string) => void;
	onSetDemo: (value: boolean) => void;
	onSetTestnet: (value: boolean) => void;
	onSetBybitRegion: (value: BybitRegion) => void;
	onSetBybitRegionEnabled: (value: boolean) => void;
	onReturnToPicker: () => void;
	onAcceptAddNotice: () => void;
	onSubmit: () => void;
	hlOverlay: HyperliquidOverlayState;
};

export const AccountOverlay = ({
	overlayMode,
	enabledClients,
	selectedExchange,
	editAccountID,
	editAccountName,
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
	onClose,
	onSetApiKey,
	onSetApiSecret,
	onSetMinimumEquity,
	onSetDemo,
	onSetTestnet,
	onSetBybitRegion,
	onSetBybitRegionEnabled,
	onReturnToPicker,
	onAcceptAddNotice,
	onSubmit,
	hlOverlay,
}: AccountOverlayProps) => {
	if (overlayMode === 'closed') return null;

	if (overlayMode === 'picker') {
		return (
			<AccountOverlayPicker
				enabledClients={enabledClients}
				onPickExchange={onPickExchange}
				onClose={onClose}
			/>
		);
	}

	if (overlayMode === 'addNotice') {
		return (
			<AccountOverlayNotice
				formError={formError}
				onReturnToPicker={onReturnToPicker}
				onAcceptAddNotice={onAcceptAddNotice}
				onClose={onClose}
			/>
		);
	}

	const isHyperliquid = selectedExchange === Exchange.Hyperliquid;
	const submitting = isHyperliquid ? hlOverlay.formSubmitting : formSubmitting;

	return (
		<ModalBackdrop
			onClose={onClose}
			closeOnEscape={!submitting}
			closeOnBackdropClick={false}
			scrollable
		>
			<AccountOverlayBody exchange={selectedExchange}>
				{isHyperliquid ? (
					<AccountOverlayForm
						exchange={Exchange.Hyperliquid}
						editAccountID={hlOverlay.editAccountID}
						editAccountName={editAccountName}
						referralURL={referralURL}
						formLoading={hlOverlay.formLoading}
						formError={hlOverlay.formError}
						formSubmitting={hlOverlay.formSubmitting}
						minimumEquity={hlOverlay.minimumEquity}
						onSetMinimumEquity={hlOverlay.onSetMinimumEquity}
						credentials={
							<AccountOverlayHyperliquidCredentials
								isEdit={!!hlOverlay.editAccountID}
								privateKey={hlOverlay.privateKey}
								masterWalletAddress={hlOverlay.masterWalletAddress}
								subaccountAddress={hlOverlay.subaccountAddress}
								onSetPrivateKey={hlOverlay.onSetPrivateKey}
								onSetMasterWalletAddress={hlOverlay.onSetMasterWalletAddress}
								onSetSubaccountAddress={hlOverlay.onSetSubaccountAddress}
							/>
						}
						options={
							<AccountOverlayHyperliquidOptions
								isDevelopMode={isDevelopMode}
								isTestNet={hlOverlay.isTestNet}
								onSetIsTestNet={hlOverlay.onSetIsTestNet}
							/>
						}
						onClose={onClose}
						onSubmit={hlOverlay.onSubmit}
					/>
				) : (
					<AccountOverlayForm
						exchange={selectedExchange}
						editAccountID={editAccountID}
						editAccountName={editAccountName}
						referralURL={referralURL}
						formLoading={formLoading}
						formError={formError}
						formSubmitting={formSubmitting}
						minimumEquity={minimumEquity}
						onSetMinimumEquity={onSetMinimumEquity}
						credentials={
							<AccountOverlayExchangeCredentials
								isEdit={!!editAccountID}
								apiKey={apiKey}
								apiSecret={apiSecret}
								onSetApiKey={onSetApiKey}
								onSetApiSecret={onSetApiSecret}
							/>
						}
						options={
							<AccountOverlayExchangeOptions
								selectedExchange={selectedExchange}
								editAccountID={editAccountID}
								isDevelopMode={isDevelopMode}
								demo={demo}
								testnet={testnet}
								bybitRegion={bybitRegion}
								bybitRegionEnabled={bybitRegionEnabled}
								onSetDemo={onSetDemo}
								onSetTestnet={onSetTestnet}
								onSetBybitRegion={onSetBybitRegion}
								onSetBybitRegionEnabled={onSetBybitRegionEnabled}
							/>
						}
						onClose={onClose}
						onSubmit={onSubmit}
					/>
				)}
			</AccountOverlayBody>
		</ModalBackdrop>
	);
};
