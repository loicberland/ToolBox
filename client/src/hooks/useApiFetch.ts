import { useState, useEffect } from 'react';
import { apiFetch } from '../utils/api';
import { API_BASE_URL } from '../config/apiConfig';

// Hook personnalisé pour les appels d'API
export const useApiFetch = <T, B = unknown>(endpoint: string, shouldFetchOnInit: boolean = true, method: string = 'GET', body?: B) => {
    const url = `${API_BASE_URL}${endpoint}`
    const [data, setData] = useState<T | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [error, setError] = useState<string | null>(null);

    const fetchData = async () => {
        setLoading(true);
        setError(null); // Réinitialiser les erreurs
        try {
            const responseData = await apiFetch<T, B>(url, method, body);
            setData(responseData);
        } catch (err) {
            setError('Erreur lors de la requête');
        } finally {
            setLoading(false);
        }
    };

    // Effectuer l'appel automatique si demandé
    useEffect(() => {
        if (shouldFetchOnInit) {
            fetchData();
        }
    }, [url, method, body, shouldFetchOnInit]); // Dépendances pour l'effet

    return { data, loading, error, fetchData }; // Retourner la fonction fetchData
};
