import { truncateHyperliquidAccountID } from '../lib/accountDisplayName';
import { AccountsSectionRenameForm } from './AccountsSectionRenameForm';
import { ExchangeLogo } from './ExchangeLogo';

type AccountsSectionItemProps = {
	account: AccountInfo;
	label: string | undefined;
	isRenaming: boolean;
	onStartRename: () => void;
	onCloseRename: () => void;
	onRename: (id: string, label: string) => Promise<void>;
	onEdit: () => void;
	onRemove: () => void;
};

export const AccountsSectionItem = ({
	account,
	label,
	isRenaming,
	onStartRename,
	onCloseRename,
	onRename,
	onEdit,
	onRemove,
}: AccountsSectionItemProps) => (
	<li className='account-item'>
		<ExchangeLogo exchange={account.ExchangeType} />
		{isRenaming ? (
			<AccountsSectionRenameForm
				accountID={account.ID}
				initialValue={label ?? ''}
				onRename={onRename}
				onClose={onCloseRename}
			/>
		) : (
			<>
				<span
					className='account-name'
					title={`${account.ID} · Click to rename`}
					onClick={onStartRename}
				>
					{label ?? truncateHyperliquidAccountID(account.ID)}
				</span>
				<div className='account-actions'>
					<button className='edit-btn' title='Edit credentials' onClick={onEdit}>
						&#9998;
					</button>
					<button className='remove-btn' title='Remove account' onClick={onRemove}>
						&#10005;
					</button>
				</div>
			</>
		)}
	</li>
);
