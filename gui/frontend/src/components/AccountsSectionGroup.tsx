import { AccountsSectionItem } from './AccountsSectionItem';

type AccountsSectionGroupProps = {
	title: string;
	variant: 'live' | 'demo';
	accounts: AccountInfo[];
	accountLabels: Record<string, string>;
	renamingID: string | undefined;
	onStartRename: (id: string) => void;
	onCloseRename: () => void;
	onRename: (id: string, label: string) => Promise<void>;
	onEdit: (id: string, exchange: AccountInfo['ExchangeType']) => void;
	onRemove: (id: string) => void;
};

export const AccountsSectionGroup = ({
	title,
	variant,
	accounts,
	accountLabels,
	renamingID,
	onStartRename,
	onCloseRename,
	onRename,
	onEdit,
	onRemove,
}: AccountsSectionGroupProps) => (
	<div className={`account-group account-group--${variant}`}>
		<h3 className='account-group-title'>{title}</h3>
		<ul className='account-list'>
			{accounts.map((account) => (
				<AccountsSectionItem
					key={account.ID}
					account={account}
					label={accountLabels[account.ID]}
					isRenaming={renamingID === account.ID}
					onStartRename={() => onStartRename(account.ID)}
					onCloseRename={onCloseRename}
					onRename={onRename}
					onEdit={() => onEdit(account.ID, account.ExchangeType)}
					onRemove={() => onRemove(account.ID)}
				/>
			))}
		</ul>
	</div>
);
