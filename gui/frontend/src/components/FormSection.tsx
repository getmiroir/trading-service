import { type ReactNode } from 'react';

import './Form.css';

type FormSectionProps = {
	title: string;
	children: ReactNode;
};

export const FormSection = ({ title, children }: FormSectionProps) => (
	<section className='form-section'>
		<h4 className='form-section__title'>{title}</h4>
		<div className='form-section__card'>{children}</div>
	</section>
);
