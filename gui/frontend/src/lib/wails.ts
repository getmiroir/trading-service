export const appApi = {
	getStatus: () => window.go.gui.App.GetStatus(),
	getVersion: () => window.go.gui.App.GetVersion(),
	getEnabledClients: () => window.go.gui.App.GetEnabledClients(),
	getVersionStatus: () => window.go.gui.App.GetVersionStatus(),
	checkAppUpdate: () => window.go.gui.App.CheckAppUpdate(),
	getExchangeReferralURL: (exchangeType: string) =>
		window.go.gui.App.GetExchangeReferralURL(exchangeType),
	startService: () => window.go.gui.App.StartService(),
	stopService: () => window.go.gui.App.StopService(),
	restartService: () => window.go.gui.App.RestartService(),
	installService: () => window.go.gui.App.InstallService(),
	uninstallService: () => window.go.gui.App.UninstallService(),
	login: () => window.go.gui.App.Login(),
	cancelLogin: () => window.go.gui.App.CancelLogin(),
};

export const managerApi = {
	verifyControlApiKey: () => window.go.manager.Manager.VerifyControlApiKey(),
	getLoginURL: () => window.go.manager.Manager.GetLoginURL(),
	deleteControlApiKey: () => window.go.manager.Manager.DeleteControlApiKey(),
	listAllAccounts: () => window.go.manager.Manager.ListAllAccounts(),
	getAccountParams: (exchangeType: string, accountID: string) =>
		window.go.manager.Manager.GetAccountParams(exchangeType, accountID),
	addAccount: (exchangeType: string, params: ExchangeAccountParams, replace: boolean) =>
		window.go.manager.Manager.AddAccount(exchangeType, params, replace),
	editAccount: (exchangeType: string, params: ExchangeAccountParams) =>
		window.go.manager.Manager.EditAccount(exchangeType, params),
	removeAccount: (accountID: string) => window.go.manager.Manager.RemoveAccount(accountID),
	isAccountMirroring: (accountID: string) =>
		window.go.manager.Manager.IsAccountMirroring(accountID),
	addHyperliquidAccount: (params: HyperliquidAccountParams, replace: boolean) =>
		window.go.manager.Manager.AddHyperliquidAccount(params, replace),
	getHyperliquidAccountParams: (accountID: string) =>
		window.go.manager.Manager.GetHyperliquidAccountParams(accountID),
	editHyperliquidAccount: (accountID: string, params: HyperliquidAccountParams) =>
		window.go.manager.Manager.EditHyperliquidAccount(accountID, params),
	prepareSupportLogs: (maxFiles?: number) =>
		window.go.manager.Manager.PrepareSupportLogs(maxFiles ?? null),
	submitReport: (message: string, withLogs: boolean, maxFiles?: number) =>
		window.go.manager.Manager.SubmitReport(message, withLogs, maxFiles ?? null),
	getAccountLabels: () => window.go.manager.Manager.GetAccountLabels(),
	setAccountLabel: (accountID: string, label: string) =>
		window.go.manager.Manager.SetAccountLabel(accountID, label),
};

export const events = {
	on: <T>(eventName: string, cb: (payload: T) => void) =>
		window.runtime.EventsOn(eventName, (payload) => cb(payload as T)),
};

// Opens a URL in the user's default browser via the Wails runtime.
export const openExternal = (url: string) => window.runtime.BrowserOpenURL(url);
