import { useCallback, useEffect, useRef, useState } from 'react';

import { AccountOverlayNoticeActionButtons } from './AccountOverlayNoticeActionButtons';
import { AccountOverlayNoticeBody } from './AccountOverlayNoticeBody';
import { AccountOverlayNoticeScrollHint } from './AccountOverlayNoticeScrollHint';
import { FormAlert } from './FormAlert';
import { Modal } from './Modal';
import { ExclamationTriangleIcon } from './icons/ExclamationTriangleIcon';

type AccountOverlayNoticeProps = {
	formError: string;
	onReturnToPicker: () => void;
	onAcceptAddNotice: () => void;
	onClose: () => void;
};

export const AccountOverlayNotice = ({
	formError,
	onReturnToPicker,
	onAcceptAddNotice,
	onClose,
}: AccountOverlayNoticeProps) => {
	const noticeBodyRef = useRef<HTMLDivElement | null>(null);
	const [noticeReachedBottomOnce, setNoticeReachedBottomOnce] = useState(false);
	const [noticeOverflows, setNoticeOverflows] = useState(false);

	const updateNoticeProgress = useCallback(() => {
		const el = noticeBodyRef.current;
		if (!el) return;
		const overflows = el.scrollHeight > el.clientHeight + 1;
		const atBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 1;
		setNoticeOverflows(overflows);
		setNoticeReachedBottomOnce((prev) => prev || atBottom);
	}, []);

	useEffect(() => {
		const id = window.requestAnimationFrame(() => {
			noticeBodyRef.current?.scrollTo({ top: 0 });
			setNoticeReachedBottomOnce(false);
			setNoticeOverflows(false);
			updateNoticeProgress();
		});
		return () => window.cancelAnimationFrame(id);
	}, [updateNoticeProgress]);

	const scrollHintVisible = noticeOverflows && !noticeReachedBottomOnce;

	return (
		<Modal
			size='lg'
			tone='warning'
			icon={<ExclamationTriangleIcon />}
			title='Important notice regarding this account'
			onClose={onClose}
			closeOnBackdropClick={false}
		>
			<AccountOverlayNoticeBody ref={noticeBodyRef} onScroll={updateNoticeProgress} />
			<AccountOverlayNoticeScrollHint visible={scrollHintVisible} />
			<FormAlert message={formError} />
			<AccountOverlayNoticeActionButtons
				acceptDisabled={!noticeReachedBottomOnce}
				onReturn={onReturnToPicker}
				onAccept={onAcceptAddNotice}
			/>
		</Modal>
	);
};
