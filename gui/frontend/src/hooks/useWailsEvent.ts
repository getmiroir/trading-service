import { useEffect } from 'react';

import { events } from '../lib/wails';

export const useWailsEvent = <T = ServiceStatusInfo>(
	eventName: string,
	handler: (payload: T) => void,
) => {
	useEffect(() => {
		const unsubscribe = events.on<T>(eventName, handler);
		return () => unsubscribe();
	}, [eventName, handler]);
};
