import { BybitRegion, bybitRegionLabel } from '../types';
import { FormField } from './FormField';
import { SecretInput } from './SecretInput';
import { SegmentedControl, type SegmentedControlOption } from './SegmentedControl';

const MINIMUM_EQUITY_HINT =
	'If your account equity falls below this amount, Miroir closes all mirrored positions.';

type AccountApiKeyFieldProps = {
	value: string;
	onChange: (value: string) => void;
	placeholder: string;
};

export const AccountApiKeyField = ({ value, onChange, placeholder }: AccountApiKeyFieldProps) => (
	<FormField label='API key' htmlFor='apiKeyInput'>
		<input
			id='apiKeyInput'
			className='form-input'
			type='text'
			value={value}
			placeholder={placeholder}
			autoComplete='off'
			spellCheck={false}
			onChange={(e) => onChange(e.target.value)}
		/>
	</FormField>
);

type AccountApiSecretFieldProps = {
	value: string;
	onChange: (value: string) => void;
	placeholder: string;
	label?: string;
};

export const AccountApiSecretField = ({
	value,
	onChange,
	placeholder,
	label = 'API secret',
}: AccountApiSecretFieldProps) => (
	<FormField label={label} htmlFor='apiSecretInput'>
		<SecretInput
			id='apiSecretInput'
			value={value}
			placeholder={placeholder}
			onChange={onChange}
		/>
	</FormField>
);

type AccountWalletAddressFieldProps = {
	id: string;
	label: string;
	value: string;
	onChange: (value: string) => void;
	optional?: boolean;
};

export const AccountWalletAddressField = ({
	id,
	label,
	value,
	onChange,
	optional,
}: AccountWalletAddressFieldProps) => (
	<FormField label={label} htmlFor={id} optional={optional}>
		<input
			id={id}
			className='form-input'
			type='text'
			value={value}
			placeholder='0x...'
			autoComplete='off'
			spellCheck={false}
			onChange={(e) => onChange(e.target.value)}
		/>
	</FormField>
);

type AccountMinimumEquityFieldProps = {
	value: string;
	onChange: (value: string) => void;
};

export const AccountMinimumEquityField = ({ value, onChange }: AccountMinimumEquityFieldProps) => (
	<FormField label='Minimum equity' htmlFor='minEquityInput' hint={MINIMUM_EQUITY_HINT}>
		<div className='form-input-group'>
			<input
				id='minEquityInput'
				className='form-input'
				type='number'
				min='0'
				step='any'
				value={value}
				placeholder='e.g. 50'
				onChange={(e) => onChange(e.target.value)}
			/>
			<span className='form-input-group__suffix'>USDT</span>
		</div>
	</FormField>
);

const regionalBybitRegions: BybitRegion[] = ['tr', 'kz'];

const bybitRegionOptions: SegmentedControlOption<BybitRegion>[] = regionalBybitRegions.map(
	(region) => ({ value: region, label: bybitRegionLabel(region) }),
);

type AccountBybitRegionFieldProps = {
	value: BybitRegion;
	onChange: (value: BybitRegion) => void;
};

export const AccountBybitRegionField = ({ value, onChange }: AccountBybitRegionFieldProps) => (
	<SegmentedControl
		name='bybitRegion'
		ariaLabel='Bybit region'
		options={bybitRegionOptions}
		value={value}
		onChange={onChange}
	/>
);
