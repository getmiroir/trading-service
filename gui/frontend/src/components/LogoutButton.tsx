import { ModalButton } from './ModalButton';

type LogoutButtonProps = {
	busy: boolean;
	onLogout: () => void;
};

export const LogoutButton = ({ busy, onLogout }: LogoutButtonProps) => (
	<ModalButton variant='danger' disabled={busy} onClick={onLogout}>
		{busy ? 'Logging out…' : 'Log out'}
	</ModalButton>
);
