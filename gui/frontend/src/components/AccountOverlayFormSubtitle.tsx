import { type ExchangeType, exchangeLabel } from '../types';

type AccountOverlayFormSubtitleProps = {
	exchange: ExchangeType;
	editAccountName?: string;
	referralURL?: string;
};

export const AccountOverlayFormSubtitle = ({
	exchange,
	editAccountName,
	referralURL,
}: AccountOverlayFormSubtitleProps) => {
	if (editAccountName) {
		return (
			<>
				Editing <strong>{editAccountName}</strong>
			</>
		);
	}

	if (!referralURL) return null;

	return (
		<>
			No account yet?{' '}
			<a
				className='modal__subtitle-link'
				href={referralURL}
				rel='noreferrer'
				onClick={(e) => {
					e.preventDefault();
					void window.runtime.BrowserOpenURL(referralURL);
				}}
			>
				Sign up on {exchangeLabel(exchange)} ›
			</a>
		</>
	);
};
