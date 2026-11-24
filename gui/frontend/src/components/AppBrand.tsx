import miroirLogo from '../assets/miroir_lockup_white.svg';
import './AppBrand.css';

type AppBrandProps = {
	size?: 'large' | 'compact';
};

export const AppBrand = ({ size = 'large' }: AppBrandProps) => {
	return (
		<div className={`app-brand app-brand--${size}`}>
			<img src={miroirLogo} alt='Miroir' className='app-brand-logo' />
		</div>
	);
};
