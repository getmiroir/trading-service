import { useCallback, useEffect, useState } from 'react';

import { appApi, managerApi } from '../lib/wails';
import { BybitRegion, Exchange, ExchangeType, OverlayMode } from '../types';

const parseBybitRegion = (value: string | undefined): BybitRegion => {
	if (value === 'tr' || value === 'kz') {
		return value;
	}
	return '';
};

type AccountFormCallbacks = {
	onAdded?: () => void;
	onRemoved?: () => void;
	onHLOpenAdd?: () => void;
	onHLOpenEdit?: (id: string) => Promise<void>;
	enabledClients?: ExchangeType[];
};

export function useAccountForm({
	onAdded,
	onRemoved,
	onHLOpenAdd,
	onHLOpenEdit,
	enabledClients,
}: AccountFormCallbacks = {}) {
	const [accounts, setAccounts] = useState<AccountInfo[]>([]);
	const [overlayMode, setOverlayMode] = useState<OverlayMode>('closed');
	const [selectedExchange, setSelectedExchange] = useState<ExchangeType | undefined>();
	const [editAccountID, setEditAccountID] = useState<string | undefined>();
	const [formLoading, setFormLoading] = useState(false);
	const [formError, setFormError] = useState('');
	const [formSubmitting, setFormSubmitting] = useState(false);
	const [apiKey, setApiKey] = useState('');
	const [apiSecret, setApiSecret] = useState('');
	const [minimumEquity, setMinimumEquity] = useState('');
	const [demo, setDemoState] = useState(false);
	const [testnet, setTestnetState] = useState(false);

	const setTestnet = useCallback((value: boolean) => {
		setTestnetState(value);
		if (value) {
			setDemoState(true);
		}
	}, []);

	const setDemo = useCallback(
		(value: boolean) => {
			if (!value && testnet) {
				return;
			}
			setDemoState(value);
		},
		[testnet],
	);
	const [bybitRegion, setBybitRegion] = useState<BybitRegion>('');
	const [bybitRegionEnabled, setBybitRegionEnabled] = useState(false);
	const [error, setError] = useState('');
	const [pendingRemoveID, setPendingRemoveID] = useState<string | undefined>();
	const [removalBlockedAccountID, setRemovalBlockedAccountID] = useState<string | undefined>();
	const [referralURLs, setReferralURLs] = useState<Partial<Record<ExchangeType, string>>>({});

	const loadAccounts = useCallback(async () => {
		try {
			const next = await managerApi.listAllAccounts();
			setAccounts(next ?? []);
		} catch (err) {
			setError(String(err));
		}
	}, []);

	useEffect(() => {
		void loadAccounts();
	}, [loadAccounts]);

	useEffect(() => {
		let cancelled = false;

		const loadReferralURLs = async () => {
			try {
				const [binance, bybit] = await Promise.all([
					appApi.getExchangeReferralURL(Exchange.Binance),
					appApi.getExchangeReferralURL(Exchange.Bybit),
				]);
				if (!cancelled) {
					setReferralURLs({ binance, bybit });
				}
			} catch {
				if (!cancelled) {
					setReferralURLs({});
				}
			}
		};

		void loadReferralURLs();

		return () => {
			cancelled = true;
		};
	}, []);

	const openAddPicker = useCallback(() => {
		setFormError('');
		setSelectedExchange(undefined);
		setEditAccountID(undefined);
		setOverlayMode('picker');
	}, []);

	const openAddForm = useCallback(
		(exchange: ExchangeType) => {
			if (enabledClients && !enabledClients.includes(exchange)) {
				return;
			}
			setSelectedExchange(exchange);
			setEditAccountID(undefined);
			setApiKey('');
			setApiSecret('');
			setMinimumEquity('');
			setDemo(false);
			setTestnet(false);
			setBybitRegion('');
			setBybitRegionEnabled(false);
			setFormLoading(false);
			setFormError('');
			setOverlayMode('addNotice');
			if (exchange === Exchange.Hyperliquid) {
				onHLOpenAdd?.();
			}
		},
		[enabledClients, onHLOpenAdd, setDemo, setTestnet],
	);

	const acceptAddNotice = useCallback(() => {
		if (!selectedExchange) return;
		setFormError('');
		setOverlayMode('addForm');
	}, [selectedExchange]);

	const openEditForm = useCallback(
		async (accountID: string, exchange: ExchangeType) => {
			setSelectedExchange(exchange);
			setEditAccountID(accountID);
			setFormError('');
			setOverlayMode('editForm');
			if (exchange === Exchange.Hyperliquid) {
				await onHLOpenEdit?.(accountID);
				return;
			}
			setFormLoading(true);
			setBybitRegion('');
			setBybitRegionEnabled(false);
			try {
				const result = await managerApi.getAccountParams(exchange, accountID);
				const account = Array.isArray(result) ? result[0] : result;
				if (Array.isArray(result) && result[1]) {
					setFormError(String(result[1]));
					return;
				}
				setApiKey(account.APIKey ?? '');
				setApiSecret(account.APISecret ?? '');
				setMinimumEquity(String(account.MinimumEquity ?? ''));
				setDemo(Boolean(account.Demo));
				setTestnet(Boolean(account.Testnet));
				setBybitRegion(parseBybitRegion(account.Region));
				setBybitRegionEnabled(parseBybitRegion(account.Region) !== '');
			} catch (err) {
				setFormError(`Failed to load account: ${String(err)}`);
			} finally {
				setFormLoading(false);
			}
		},
		[onHLOpenEdit, setDemo, setTestnet],
	);

	const closeOverlay = useCallback(() => {
		setOverlayMode('closed');
		setFormLoading(false);
		setFormSubmitting(false);
	}, []);

	const submitAccount = useCallback(async () => {
		if (!selectedExchange) return;
		if (!apiKey.trim() || !apiSecret.trim()) {
			setFormError('Both API key and secret are required.');
			return;
		}
		if (selectedExchange === Exchange.Bybit && bybitRegionEnabled && bybitRegion === '') {
			setFormError('Please select a region.');
			return;
		}
		setFormSubmitting(true);
		setFormError('');
		try {
			const params: ExchangeAccountParams = {
				APIKey: apiKey.trim(),
				APISecret: apiSecret.trim(),
				Demo: demo,
			};
			const parsed = Number.parseFloat(minimumEquity);
			if (!Number.isNaN(parsed)) {
				params.MinimumEquity = parsed;
			}
			if (selectedExchange === Exchange.Bybit) {
				params.Region = bybitRegion;
				params.Testnet = testnet;
			}
			if (editAccountID) {
				await managerApi.editAccount(selectedExchange, params);
			} else {
				await managerApi.addAccount(selectedExchange, params, false);
			}
			closeOverlay();
			await loadAccounts();
			onAdded?.();
		} catch (err) {
			setFormError(String(err));
		} finally {
			setFormSubmitting(false);
		}
	}, [
		selectedExchange,
		apiKey,
		apiSecret,
		demo,
		testnet,
		bybitRegion,
		bybitRegionEnabled,
		minimumEquity,
		editAccountID,
		closeOverlay,
		loadAccounts,
		onAdded,
	]);

	const handleRemoveAccount = useCallback(async (id: string) => {
		try {
			const mirroring = await managerApi.isAccountMirroring(id);
			if (mirroring) {
				setRemovalBlockedAccountID(id);
				return;
			}
		} catch {
			// If the check fails (e.g. no API key), fall through to the
			// normal remove confirmation rather than blocking the user.
		}
		setPendingRemoveID(id);
	}, []);

	const confirmRemove = useCallback(async () => {
		if (!pendingRemoveID) return;
		const id = pendingRemoveID;
		setPendingRemoveID(undefined);
		try {
			await managerApi.removeAccount(id);
			await loadAccounts();
			onRemoved?.();
		} catch (err) {
			setError(String(err));
		}
	}, [pendingRemoveID, loadAccounts, onRemoved]);

	const cancelRemove = useCallback(() => {
		setPendingRemoveID(undefined);
	}, []);

	const dismissAccountRemovalBlocked = useCallback(() => {
		setRemovalBlockedAccountID(undefined);
	}, []);

	return {
		loadAccounts,
		accounts,
		overlayMode,
		selectedExchange,
		editAccountID,
		formLoading,
		formError,
		formSubmitting,
		apiKey,
		apiSecret,
		minimumEquity,
		demo,
		testnet,
		bybitRegion,
		bybitRegionEnabled,
		error,
		setApiKey,
		setApiSecret,
		setMinimumEquity,
		setDemo,
		setTestnet,
		setBybitRegion,
		setBybitRegionEnabled,
		openAddPicker,
		openAddForm,
		acceptAddNotice,
		openEditForm,
		closeOverlay,
		submitAccount,
		handleRemoveAccount,
		pendingRemoveID,
		confirmRemove,
		cancelRemove,
		removalBlockedAccountID,
		dismissAccountRemovalBlocked,
		referralURLs,
	};
}
