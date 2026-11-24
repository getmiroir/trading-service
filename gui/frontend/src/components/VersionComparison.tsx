type VersionComparisonRow = {
	label: string;
	value: string;
	variant: 'outgoing' | 'incoming';
};

type VersionComparisonProps = {
	rows: VersionComparisonRow[];
};

export const VersionComparison = ({ rows }: VersionComparisonProps) => (
	<div className='version-comparison'>
		{rows.map((row) => (
			<div key={row.label} className='version-comparison__row'>
				<span className='version-comparison__label'>{row.label}</span>
				<span
					className={`version-comparison__value version-comparison__value--${row.variant}`}
				>
					{row.value}
				</span>
			</div>
		))}
	</div>
);
