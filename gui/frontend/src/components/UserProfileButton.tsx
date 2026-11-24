type UserProfileButtonProps = {
	onClick: () => void;
};

export const UserProfileButton = ({ onClick }: UserProfileButtonProps) => (
	<button
		className='profile-btn'
		type='button'
		onClick={onClick}
		title='Settings'
		aria-label='Settings'
	>
		<span className='profile-btn__gear' aria-hidden={true}>
			⚙
		</span>
	</button>
);
