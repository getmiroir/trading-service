import { BybitRegion } from '../types';
import { AccountBybitRegionField } from './FormElements';
import { Switch } from './Switch';

type AccountOverlayBybitRegionSectionProps = {
	bybitRegion: BybitRegion;
	bybitRegionEnabled: boolean;
	onSetBybitRegion: (value: BybitRegion) => void;
	onSetBybitRegionEnabled: (value: boolean) => void;
};

export const AccountOverlayBybitRegionSection = ({
	bybitRegion,
	bybitRegionEnabled,
	onSetBybitRegion,
	onSetBybitRegionEnabled,
}: AccountOverlayBybitRegionSectionProps) => {
	const expanded = bybitRegionEnabled || bybitRegion !== '';

	const handleToggle = (checked: boolean) => {
		onSetBybitRegionEnabled(checked);
		if (!checked) {
			onSetBybitRegion('');
		}
	};

	return (
		<>
			<Switch
				id='bybitRegionSwitch'
				label='Registered in Turkey or Kazakhstan'
				checked={expanded}
				onChange={handleToggle}
			/>
			{expanded && (
				<AccountBybitRegionField value={bybitRegion} onChange={onSetBybitRegion} />
			)}
		</>
	);
};
