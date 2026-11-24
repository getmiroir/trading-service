export const buildAccountDeepLink = (
	loginURL: string | undefined,
	accountID: string,
): string | undefined => {
	if (!loginURL) {
		return undefined;
	}
	try {
		const url = new URL(loginURL);
		const basePath = url.pathname.replace(/\/login\/?$/, '').replace(/\/+$/, '');
		url.pathname = `${basePath}/accounts/${encodeURIComponent(accountID)}`;
		url.search = '';
		url.hash = '';
		return url.toString();
	} catch {
		return undefined;
	}
};
