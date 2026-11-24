import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

import binanceDemoMd from '../tutorials/binance_demo.md?raw';
import binanceProdMd from '../tutorials/binance_prod.md?raw';
import binanceRefMd from '../tutorials/binance_ref.md?raw';
import bybitDemoMd from '../tutorials/bybit_demo.md?raw';
import bybitProdMd from '../tutorials/bybit_prod.md?raw';
import bybitRefMd from '../tutorials/bybit_ref.md?raw';
import hyperliquidProdMd from '../tutorials/hyperliquid_prod.md?raw';
import hyperliquidRefMd from '../tutorials/hyperliquid_ref.md?raw';
import { ExchangeType, exchangeLabel } from '../types';
import { BookIcon } from './icons/BookIcon';

const tutorials: Record<ExchangeType, { prod: string; demo?: string; ref: string }> = {
	binance: { prod: binanceProdMd, demo: binanceDemoMd, ref: binanceRefMd },
	bybit: { prod: bybitProdMd, demo: bybitDemoMd, ref: bybitRefMd },
	hyperliquid: { prod: hyperliquidProdMd, ref: hyperliquidRefMd },
};

type TutorialPanelProps = {
	exchange: ExchangeType;
};

const mdComponents: React.ComponentProps<typeof ReactMarkdown>['components'] = {
	a: ({ href, children, ...props }) => (
		<a
			{...props}
			className='app-link'
			href={href}
			rel='noreferrer'
			onClick={(e) => {
				if (!href) return;
				e.preventDefault();
				void window.runtime.BrowserOpenURL(href);
			}}
		>
			{children}
		</a>
	),
};

type SectionProps = {
	title: string;
	content: string;
};

const TutorialSection = ({ title, content }: SectionProps) => (
	<details className='tutorial-panel__section'>
		<summary className='tutorial-panel__summary'>
			<span>{title}</span>
			<span className='tutorial-panel__chevron' aria-hidden='true'>
				›
			</span>
		</summary>
		<div className='tutorial-panel__body'>
			<ReactMarkdown remarkPlugins={[remarkGfm]} components={mdComponents}>
				{content}
			</ReactMarkdown>
		</div>
	</details>
);

export const TutorialPanel = ({ exchange }: TutorialPanelProps) => (
	<div className='tutorial-panel'>
		<div className='tutorial-panel__header'>
			<BookIcon className='tutorial-panel__header-icon' aria-hidden='true' />
			<span>{exchangeLabel(exchange)} setup guides</span>
		</div>
		<TutorialSection title='How to create API keys' content={tutorials[exchange].prod} />
		{tutorials[exchange].demo && (
			<TutorialSection
				title='How to trade with a demo account'
				content={tutorials[exchange].demo}
			/>
		)}
		<TutorialSection title='Quick reference' content={tutorials[exchange].ref} />
	</div>
);
