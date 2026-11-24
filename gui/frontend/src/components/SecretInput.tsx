import { useState } from 'react';

import './Form.css';
import { EyeIcon } from './icons/EyeIcon';
import { EyeSlashIcon } from './icons/EyeSlashIcon';

type SecretInputProps = {
	id: string;
	value: string;
	placeholder: string;
	onChange: (value: string) => void;
};

export const SecretInput = ({ id, value, placeholder, onChange }: SecretInputProps) => {
	const [revealed, setRevealed] = useState(false);

	return (
		<div className='form-input-group'>
			<input
				id={id}
				className='form-input'
				type={revealed ? 'text' : 'password'}
				value={value}
				placeholder={placeholder}
				autoComplete='off'
				spellCheck={false}
				onChange={(e) => onChange(e.target.value)}
			/>
			<button
				type='button'
				className='form-input-group__action'
				aria-label={revealed ? 'Hide value' : 'Show value'}
				aria-pressed={revealed}
				onClick={() => setRevealed((prev) => !prev)}
			>
				{revealed ? <EyeSlashIcon /> : <EyeIcon />}
			</button>
		</div>
	);
};
