export const isDevelopVersion = (version: string) => {
	const lower = version.toLowerCase();
	return lower === 'develop';
};
