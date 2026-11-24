import { useState } from 'react';

import { truncateHyperliquidAccountID } from '../lib/accountDisplayName';

type AccountsSectionRenameFormProps = {
	accountID: string;
	initialValue: string;
	onRename: (id: string, label: string) => Promise<void>;
	onClose: () => void;
};

export const AccountsSectionRenameForm = ({
	accountID,
	initialValue,
	onRename,
	onClose,
}: AccountsSectionRenameFormProps) => {
	const [value, setValue] = useState(initialValue);
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState('');

	const save = async () => {
		setSaving(true);
		setError('');
		try {
			await onRename(accountID, value.trim());
			onClose();
		} catch {
			setError('Could not rename the account. Please check your connection.');
			setSaving(false);
		}
	};

	return (
		<>
			<input
				className='account-rename-input'
				type='text'
				value={value}
				maxLength={64}
				placeholder={truncateHyperliquidAccountID(accountID)}
				disabled={saving}
				onChange={(e) => setValue(e.target.value)}
				onKeyDown={(e) => {
					if (e.key === 'Enter') void save();
					if (e.key === 'Escape') onClose();
				}}
				autoFocus
			/>
			{error && <span className='account-rename-error'>{error}</span>}
			<div className='account-actions'>
				<button
					className='edit-btn'
					title='Save name'
					disabled={saving}
					onClick={() => void save()}
				>
					✓
				</button>
				<button className='remove-btn' title='Cancel' disabled={saving} onClick={onClose}>
					✕
				</button>
			</div>
		</>
	);
};
