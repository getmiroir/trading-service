import { type ReactNode } from 'react';

import { type ExchangeType } from '../types';
import { TutorialPanel } from './TutorialPanel';

type AccountOverlayBodyProps = {
	children: ReactNode;
	exchange?: ExchangeType;
};

export const AccountOverlayBody = ({ children, exchange }: AccountOverlayBodyProps) => (
	<div className='overlay-body'>
		<div className='overlay-body__form'>{children}</div>
		{exchange ? (
			<>
				<div className='overlay-body__tutorial overlay-body__tutorial--desktop'>
					<TutorialPanel exchange={exchange} />
				</div>
				<div className='overlay-body__tutorial overlay-body__tutorial--mobile'>
					<TutorialPanel exchange={exchange} />
				</div>
			</>
		) : null}
	</div>
);
