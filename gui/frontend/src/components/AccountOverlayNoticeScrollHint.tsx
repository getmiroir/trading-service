type AccountOverlayNoticeScrollHintProps = {
	visible: boolean;
};

export const AccountOverlayNoticeScrollHint = ({
	visible,
}: AccountOverlayNoticeScrollHintProps) => (
	<p
		className={
			visible
				? 'notice-scroll-hint notice-scroll-hint--visible'
				: 'notice-scroll-hint notice-scroll-hint--hidden'
		}
	>
		Scroll to the bottom to continue.
	</p>
);
