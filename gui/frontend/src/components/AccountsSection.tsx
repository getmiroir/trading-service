import { useState } from 'react';

import { AccountsSectionGroup } from './AccountsSectionGroup';

type AccountsSectionProps = {
	accounts: AccountInfo[];
	accountLabels: Record<string, string>;
	onAdd: () => void;
	onEdit: (id: string, exchange: AccountInfo['ExchangeType']) => void;
	onRemove: (id: string) => void;
	onRename: (id: string, label: string) => Promise<void>;
};

export const AccountsSection = ({
	accounts,
	accountLabels,
	onAdd,
	onEdit,
	onRemove,
	onRename,
}: AccountsSectionProps) => {
	const [renamingID, setRenamingID] = useState<string | undefined>();

	const liveAccounts = accounts.filter((account) => !account.Demo);
	const demoAccounts = accounts.filter((account) => account.Demo);

	const groupProps = {
		accountLabels,
		renamingID,
		onStartRename: setRenamingID,
		onCloseRename: () => setRenamingID(undefined),
		onRename,
		onEdit,
		onRemove,
	};

	return (
		<div className='accounts-section'>
			<div className='accounts-header'>
				<h2>Accounts</h2>
				<button className='add-btn' title='Add account' onClick={onAdd}>
					+
				</button>
			</div>
			<div className='account-list-scroll'>
				{accounts.length === 0 ? (
					<ul className='account-list'>
						<li className='no-accounts'>No accounts configured</li>
					</ul>
				) : (
					<>
						{liveAccounts.length > 0 && (
							<AccountsSectionGroup
								title='Live'
								variant='live'
								accounts={liveAccounts}
								{...groupProps}
							/>
						)}
						{demoAccounts.length > 0 && (
							<AccountsSectionGroup
								title='Demo'
								variant='demo'
								accounts={demoAccounts}
								{...groupProps}
							/>
						)}
					</>
				)}
			</div>
		</div>
	);
};
