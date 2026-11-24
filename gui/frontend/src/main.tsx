import React from 'react';
import ReactDOM from 'react-dom/client';

import { WailsApp } from './WailsApp';
import './styles.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
	<React.StrictMode>
		<WailsApp />
	</React.StrictMode>,
);
