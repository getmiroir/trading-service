import './Form.css';

export type SegmentedControlOption<T extends string> = {
	value: T;
	label: string;
};

type SegmentedControlProps<T extends string> = {
	name: string;
	ariaLabel: string;
	options: SegmentedControlOption<T>[];
	value: T;
	onChange: (value: T) => void;
};

export const SegmentedControl = <T extends string>({
	name,
	ariaLabel,
	options,
	value,
	onChange,
}: SegmentedControlProps<T>) => (
	<div className='segmented' role='radiogroup' aria-label={ariaLabel}>
		{options.map((option) => (
			<label key={option.value} className='segmented__option'>
				<input
					className='segmented__input'
					type='radio'
					name={name}
					value={option.value}
					checked={option.value === value}
					onChange={() => onChange(option.value)}
				/>
				<span className='segmented__label'>{option.label}</span>
			</label>
		))}
	</div>
);
