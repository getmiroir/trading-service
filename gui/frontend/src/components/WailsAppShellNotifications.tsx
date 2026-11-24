import { type ReactNode, forwardRef } from 'react';

type WailsAppShellNotificationsProps = {
	children: ReactNode;
};

export const WailsAppShellNotifications = forwardRef<
	HTMLDivElement,
	WailsAppShellNotificationsProps
>(function WailsAppShellNotifications({ children }, ref) {
	return (
		<>
			<h2 className='app-shell-notifications__title'>Notifications</h2>
			<div
				ref={ref}
				className='app-shell-notifications'
				role='region'
				aria-label='Notifications'
			>
				{children}
			</div>
		</>
	);
});
