import { useEffect, useState } from 'react';

import { ModalButton } from './ModalButton';

type ProfileServiceActionsProps = {
	installVisible: boolean;
	installDisabled: boolean;
	uninstallVisible: boolean;
	uninstallDisabled: boolean;
	uninstallShowBlockedTooltip: boolean;
	onInstall: () => void;
	onUninstall: () => void;
};

type SuppressedSnapshot = {
	installVisible: boolean;
	uninstallVisible: boolean;
	uninstallDisabled: boolean;
} | null;

export const ProfileServiceActions = ({
	installVisible,
	installDisabled,
	uninstallVisible,
	uninstallDisabled,
	uninstallShowBlockedTooltip,
	onInstall,
	onUninstall,
}: ProfileServiceActionsProps) => {
	const [suppressedSnapshot, setSuppressedSnapshot] = useState<SuppressedSnapshot>(null);

	// Suppression is only active when props still match the snapshot taken at click time.
	const suppressedBlockedTooltip =
		suppressedSnapshot !== null &&
		suppressedSnapshot.installVisible === installVisible &&
		suppressedSnapshot.uninstallVisible === uninstallVisible &&
		suppressedSnapshot.uninstallDisabled === uninstallDisabled;

	useEffect(() => {
		if (!suppressedBlockedTooltip) return;
		const clear = () => setSuppressedSnapshot(null);
		window.addEventListener('pointermove', clear, { once: true });
		window.addEventListener('keydown', clear, { once: true });
		return () => {
			window.removeEventListener('pointermove', clear);
			window.removeEventListener('keydown', clear);
		};
	}, [suppressedBlockedTooltip]);

	if (installVisible) {
		return (
			<ModalButton variant='primary' disabled={installDisabled} onClick={onInstall}>
				Install Service
			</ModalButton>
		);
	}
	if (uninstallVisible) {
		const uninstallButton = (
			<ModalButton
				variant='danger'
				disabled={uninstallDisabled}
				onClick={() => {
					if (!uninstallDisabled) {
						setSuppressedSnapshot({
							installVisible,
							uninstallVisible,
							uninstallDisabled,
						});
					}
					onUninstall();
				}}
			>
				Uninstall Service
			</ModalButton>
		);
		return uninstallShowBlockedTooltip ? (
			<span
				className={
					suppressedBlockedTooltip
						? 'profile-service-actions__uninstall-blocked profile-service-actions__uninstall-blocked--suppress-tooltip'
						: 'profile-service-actions__uninstall-blocked'
				}
				data-tooltip='Turn the service off before uninstalling.'
			>
				{uninstallButton}
			</span>
		) : (
			uninstallButton
		);
	}
	return null;
};
