import './Form.css';
import { ExclamationTriangleIcon } from './icons/ExclamationTriangleIcon';

type FormAlertProps = {
	message: string;
};

export const FormAlert = ({ message }: FormAlertProps) => {
	if (!message) return null;

	return (
		<div className='form-alert' role='alert'>
			<ExclamationTriangleIcon className='form-alert__icon' aria-hidden='true' />
			<span>{message}</span>
		</div>
	);
};
