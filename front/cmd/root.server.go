/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

// root.serverCmd represents the root.server command
var rootServerCmd = &cobra.Command{
	Use:   "server",
	Short: "starting server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		server()
	},
}

func init() {
	rootCmd.AddCommand(rootServerCmd)
}

type HomeData struct {
	Message string `json:"message"`
}

func homeAPIHandler(w http.ResponseWriter, r *http.Request) {
	data := HomeData{
		Message: "Welcome to the Home Page!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func server() {

	port := 8080
	fqdn := "PORTLB"
	protocol := "http"
	listenURL := fmt.Sprintf("%s://%s:%d", protocol, fqdn, port)
	listenSocket := fmt.Sprintf(":%d", port)

	r := mux.NewRouter()
	r.HandleFunc("/", homeAPIHandler).Methods("GET")

	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Remplacez par l'URL de votre frontend React
		AllowCredentials: true,
	})
	fmt.Printf("Starting server at %s \n", listenURL)
	if err := http.ListenAndServe(listenSocket, c.Handler(r)); err != nil {
		fmt.Println(err)
	}
}
