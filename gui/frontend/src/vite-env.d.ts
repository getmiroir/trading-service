/// <reference types="vite/client" />

declare module '*.svg?react' {
	import type { FC, SVGProps } from 'react';
	const ReactComponent: FC<SVGProps<SVGSVGElement>>;
	export default ReactComponent;
}

type ServiceStatusInfo = { status: string };

type VersionStatus = 'ok' | 'upgrade' | 'downgrade' | 'unknown';

type VersionInfo = {
	guiVersion: string;
	serviceVersion: string;
	status: VersionStatus;
};

type AppUpdateInfo = {
	currentVersion: string;
	latestVersion: string;
	updateAvailable: boolean;
	downloadUrl: string;
};

type AccountInfo = {
	ID: string;
	ExchangeType: 'bybit' | 'binance' | 'hyperliquid';
	Demo: boolean;
};

type ExchangeAccountParams = {
	Demo?: boolean;
	Testnet?: boolean;
	APIKey: string;
	APISecret: string;
	MinimumEquity?: number;
	Region?: string;
};

type HyperliquidAccountParams = {
	IsTestNet?: boolean;
	PrivateKey: string;
	MasterWalletAddress: string;
	SubaccountAddress?: string;
	MinimumEquity?: number;
};

type VerifyResult = {
	status: 'valid' | 'missing' | 'invalid' | 'unreachable';
	auth?: {
		email?: string;
		userId?: string;
		apiKey?: {
			keyId: string;
			name: string;
			createdAt: string;
		};
	};
};

interface Window {
	go: {
		gui: {
			App: {
				StartService: () => Promise<void>;
				StopService: () => Promise<void>;
				RestartService: () => Promise<void>;
				InstallService: () => Promise<void>;
				UninstallService: () => Promise<void>;
				GetStatus: () => Promise<ServiceStatusInfo>;
				GetVersion: () => Promise<string>;
				GetEnabledClients: () => Promise<string[]>;
				GetVersionStatus: () => Promise<VersionInfo>;
				CheckAppUpdate: () => Promise<AppUpdateInfo>;
				GetExchangeReferralURL: (exchangeType: string) => Promise<string>;
				Login: () => Promise<void>;
				CancelLogin: () => Promise<void>;
			};
		};
		manager: {
			Manager: {
				VerifyControlApiKey: () => Promise<VerifyResult>;
				GetLoginURL: () => Promise<string>;
				DeleteControlApiKey: () => Promise<void>;
				ListAllAccounts: () => Promise<AccountInfo[]>;
				GetAccountParams: (
					exchangeType: string,
					accountID: string,
				) => Promise<ExchangeAccountParams | [ExchangeAccountParams, string | null]>;
				AddAccount: (
					exchangeType: string,
					params: ExchangeAccountParams,
					replace: boolean,
				) => Promise<void>;
				EditAccount: (exchangeType: string, params: ExchangeAccountParams) => Promise<void>;
				RemoveAccount: (accountID: string) => Promise<void>;
				IsAccountMirroring: (accountID: string) => Promise<boolean>;
				AddHyperliquidAccount: (
					params: HyperliquidAccountParams,
					replace: boolean,
				) => Promise<void>;
				GetHyperliquidAccountParams: (
					accountID: string,
				) => Promise<HyperliquidAccountParams | [HyperliquidAccountParams, string | null]>;
				EditHyperliquidAccount: (
					accountID: string,
					params: HyperliquidAccountParams,
				) => Promise<void>;
				PrepareSupportLogs: (
					maxFiles: number | null,
				) => Promise<{ serviceLogsSize: number; guiLogsSize: number }>;
				SubmitReport: (
					message: string,
					withLogs: boolean,
					maxFiles: number | null,
				) => Promise<{ success: boolean; message: string }>;
				GetAccountLabels: () => Promise<Record<string, string> | null>;
				SetAccountLabel: (accountID: string, label: string) => Promise<void>;
			};
		};
	};
	runtime: {
		BrowserOpenURL: (url: string) => Promise<void>;
		EventsOn: (eventName: string, cb: (payload: unknown) => void) => () => void;
	};
}
