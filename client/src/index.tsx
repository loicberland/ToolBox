import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './components/App/App';

const rootElement = document.getElementById('root'); // Trouvez l'élément 'root'
if (rootElement) {
    const root = ReactDOM.createRoot(rootElement);
    root.render(<App />); // Rendre l'application avec le nouveau root
} else {
    console.error("L'élément root n'a pas été trouvé dans le DOM.");
}
