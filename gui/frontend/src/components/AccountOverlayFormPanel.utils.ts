import { type ExchangeType, exchangeLabel } from '../types';

export const getAccountFormTitle = (exchange: ExchangeType | undefined, isEdit: boolean): string =>
	[isEdit ? 'Edit' : 'Add', exchange ? exchangeLabel(exchange) : undefined, 'account']
		.filter(Boolean)
		.join(' ');
