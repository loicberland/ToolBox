import React, { useState } from 'react';
import { useApiFetch } from '../../hooks/useApiFetch';

const App = () => {
    const { data, loading, error, fetchData } = useApiFetch<{ message: string }>('/home', false);

    return (
        <div>
            <h1>Boîte à outils</h1>
            <button onClick={fetchData}>Récupérer le message</button>
            {loading ? <p>Chargement...</p> : <p>Message: {data?.message}</p>}
        </div>
    );
};

export default App;
