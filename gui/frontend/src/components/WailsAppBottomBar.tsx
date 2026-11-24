type WailsAppBottomBarProps = {
	version: string;
	onReport: () => void;
};

export const WailsAppBottomBar = ({ version, onReport }: WailsAppBottomBarProps) => (
	<div className='app-shell__bottombar'>
		<span className='version-footer'>{version}</span>
		<button className='report-footer-btn' onClick={onReport}>
			Report an issue
		</button>
	</div>
);
