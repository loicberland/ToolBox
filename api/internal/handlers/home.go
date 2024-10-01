package handlers

import (
	"encoding/json"
	"net/http"
	"toolBox/api/internal/models"

	"github.com/gorilla/mux"
)

func HomeAPIHandler(w http.ResponseWriter, r *http.Request) {
	data := models.HomeData{
		Message: "Welcome to the Home Page!",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SetupRoutes configure toutes les routes
func SetupRoutes(r *mux.Router) {
	r.HandleFunc("/", HomeAPIHandler).Methods("GET")
	// Ajouter d'autres routes ici
}
