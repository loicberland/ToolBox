package handlers

import (
	"encoding/json"
	"net/http"
	"toolBox/api/internal/models"

	"github.com/gorilla/mux"
)

func HomeAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Vérifier le type de requête (GET, POST, etc.)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Traiter la requête (récupérer des données, effectuer des opérations...)
	data := models.HomeData{
		Message: "Welcome to the Home Page!",
	}

	// 3. Répondre au client
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SetupRoutes configure toutes les routes
func SetupRoutes(r *mux.Router) {
	r.HandleFunc("/api/home", HomeAPIHandler).Methods("GET")
	// Ajouter d'autres routes ici
}
