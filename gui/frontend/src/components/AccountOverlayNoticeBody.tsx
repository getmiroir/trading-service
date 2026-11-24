import { forwardRef } from 'react';

type AccountOverlayNoticeBodyProps = {
	onScroll: () => void;
};

export const AccountOverlayNoticeBody = forwardRef<HTMLDivElement, AccountOverlayNoticeBodyProps>(
	function AccountOverlayNoticeBody({ onScroll }, ref) {
		return (
			<div ref={ref} className='notice-body' tabIndex={0} onScroll={onScroll}>
				<p>
					You are about to connect this account to our copy trading service. Here is what
					that means and what you need to know.
				</p>
				<p>
					Copy trading is a feature that automatically mirrors the trades of an
					experienced trader on your account. This means that whenever that trader opens,
					adjusts, or closes a position, the same action is applied to your account
					automatically - without you having to do anything.
				</p>
				<p>
					Because your account is being managed automatically, it is important that you do
					not manually place, modify, or close any trades on this account yourself. If you
					do, our system will detect that your account is out of sync with the trader it
					is following and will attempt to correct it automatically. This correction
					process may trigger additional transactions, which can result in unexpected
					fees.
				</p>
				<p>
					If you ever need to take manual control of this account - for example, to
					withdraw funds or review your positions - please make sure to disable copy
					trading from the web interface first. This will pause the automatic management
					of your account and allow you to act freely without any unintended consequences.
				</p>
				<p>
					By clicking Accept, you confirm that you have read and understood the above. You
					acknowledge that any manual actions taken on this account while copy trading is
					active - and any fees or consequences that may result from them - are your sole
					responsibility.
				</p>
			</div>
		);
	},
);
