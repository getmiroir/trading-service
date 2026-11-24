import './Form.css';

type SwitchProps = {
	id: string;
	label: string;
	description?: string;
	checked: boolean;
	disabled?: boolean;
	onChange: (checked: boolean) => void;
};

export const Switch = ({ id, label, description, checked, disabled, onChange }: SwitchProps) => (
	<label className={disabled ? 'switch switch--disabled' : 'switch'} htmlFor={id}>
		<span className='switch__text'>
			<span className='switch__label'>{label}</span>
			{description && <span className='switch__description'>{description}</span>}
		</span>
		<input
			id={id}
			className='switch__input'
			type='checkbox'
			role='switch'
			checked={checked}
			disabled={disabled}
			onChange={(e) => onChange(e.target.checked)}
		/>
		<span className='switch__track' aria-hidden='true'>
			<span className='switch__thumb' />
		</span>
	</label>
);
