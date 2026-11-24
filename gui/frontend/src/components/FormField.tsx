import { type ReactNode } from 'react';

import './Form.css';

type FormFieldProps = {
	label: string;
	htmlFor: string;
	hint?: string;
	optional?: boolean;
	children: ReactNode;
};

export const FormField = ({ label, htmlFor, hint, optional, children }: FormFieldProps) => (
	<div className='form-field'>
		<label className='form-field__label' htmlFor={htmlFor}>
			{label}
			{optional && <span className='form-field__badge'>Optional</span>}
		</label>
		{children}
		{hint && <p className='form-field__hint'>{hint}</p>}
	</div>
);
