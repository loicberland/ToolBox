// Crée une fonction générique pour faire des requêtes API
export const apiFetch = async <T, B>(url: string, method: string = 'GET', body?: B): Promise<T> => {
    try {
        // Options pour la requête, avec les headers et la méthode
        const options: RequestInit = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
        };

        // Si c'est une méthode POST/PUT, on inclut le body (données) dans la requête
        if (body && (method === 'POST' || method === 'PUT')) {
            options.body = JSON.stringify(body);
        }

        // Effectuer la requête
        const response = await fetch(url, options);

        // Vérifier que la requête a réussi
        if (!response.ok) {
            throw new Error(`Erreur HTTP: ${response.status}`);
        }

        // Extraire les données JSON
        const data = await response.json();
        return data; // Retourner les données
    } catch (error) {
        console.error('Erreur lors de la requête API:', error);
        throw error; // On relance l'erreur pour la gérer dans le composant
    }
};
