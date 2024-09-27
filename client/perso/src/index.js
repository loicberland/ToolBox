import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './components/App/App'

const rootElement = document.getElementById('root'); // Trouvez l'élément 'root'
const root = ReactDOM.createRoot(rootElement); // Créez un nouveau root

root.render(
  <App />
); // Rendre l'application avec le nouveau root
