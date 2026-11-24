import { useCallback, useState } from 'react';

import { managerApi } from '../lib/wails';

type Callbacks = {
	onAdded?: () => void;
};

export function useHyperliquidAccountForm({ onAdded }: Callbacks = {}) {
	const [editAccountID, setEditAccountID] = useState<string | undefined>();
	const [formLoading, setFormLoading] = useState(false);
	const [formError, setFormError] = useState('');
	const [formSubmitting, setFormSubmitting] = useState(false);
	const [privateKey, setPrivateKey] = useState('');
	const [masterWalletAddress, setMasterWalletAddress] = useState('');
	const [subaccountAddress, setSubaccountAddress] = useState('');
	const [minimumEquity, setMinimumEquity] = useState('');
	const [isTestNet, setIsTestNet] = useState(false);

	const openAddForm = useCallback(() => {
		setEditAccountID(undefined);
		setPrivateKey('');
		setMasterWalletAddress('');
		setSubaccountAddress('');
		setMinimumEquity('');
		setIsTestNet(false);
		setFormError('');
		setFormLoading(false);
		setFormSubmitting(false);
	}, []);

	const openEditForm = useCallback(async (accountID: string) => {
		setEditAccountID(accountID);
		setFormError('');
		setFormLoading(true);
		try {
			const result = await managerApi.getHyperliquidAccountParams(accountID);
			const account = Array.isArray(result) ? result[0] : result;
			if (Array.isArray(result) && result[1]) {
				setFormError(String(result[1]));
				return;
			}
			setPrivateKey(account.PrivateKey ?? '');
			setMasterWalletAddress(account.MasterWalletAddress ?? '');
			setSubaccountAddress(account.SubaccountAddress ?? '');
			setMinimumEquity(String(account.MinimumEquity ?? ''));
			setIsTestNet(Boolean(account.IsTestNet));
		} catch (err) {
			setFormError(`Failed to load account: ${String(err)}`);
		} finally {
			setFormLoading(false);
		}
	}, []);

	const submitAccount = useCallback(async (): Promise<boolean> => {
		if (!privateKey.trim()) {
			setFormError('Private key is required.');
			return false;
		}
		if (!masterWalletAddress.trim()) {
			setFormError('Master wallet address is required.');
			return false;
		}
		setFormSubmitting(true);
		setFormError('');
		try {
			const params: HyperliquidAccountParams = {
				PrivateKey: privateKey.trim(),
				MasterWalletAddress: masterWalletAddress.trim(),
				IsTestNet: isTestNet,
			};
			const trimmedSubaccount = subaccountAddress.trim().replace(/^HL:/i, '');
			if (trimmedSubaccount) {
				params.SubaccountAddress = trimmedSubaccount;
			}
			const parsed = Number.parseFloat(minimumEquity);
			if (!Number.isNaN(parsed)) {
				params.MinimumEquity = parsed;
			}
			if (editAccountID) {
				await managerApi.editHyperliquidAccount(editAccountID, params);
			} else {
				await managerApi.addHyperliquidAccount(params, false);
			}
			onAdded?.();
			return true;
		} catch (err) {
			setFormError(String(err));
			return false;
		} finally {
			setFormSubmitting(false);
		}
	}, [
		privateKey,
		masterWalletAddress,
		subaccountAddress,
		isTestNet,
		minimumEquity,
		editAccountID,
		onAdded,
	]);

	return {
		editAccountID,
		formLoading,
		formError,
		formSubmitting,
		privateKey,
		masterWalletAddress,
		subaccountAddress,
		minimumEquity,
		isTestNet,
		setPrivateKey,
		setMasterWalletAddress,
		setSubaccountAddress,
		setMinimumEquity,
		setIsTestNet,
		openAddForm,
		openEditForm,
		submitAccount,
	};
}
