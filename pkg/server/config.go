package server

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ConfigServer struct {
	Port     int
	FQDN     string
	Protocol string
}

func LoadServerConfig(who string) (c *ConfigServer, err error) {
	// Charger le fichier .env
	errGetEnv := godotenv.Load()
	if errGetEnv != nil {
		err = fmt.Errorf("erreur de chargement du fichier .env: %s", errGetEnv)
		return
	}

	// Convertir le port de string à int
	portStr := os.Getenv("PORT_" + who)
	port, errConvert := strconv.Atoi(portStr)
	if errConvert != nil {
		err = fmt.Errorf("erreur de conversion du port : %s", errConvert)
		return
	}

	// Créer la config
	config := &ConfigServer{
		Port:     port,
		FQDN:     os.Getenv("FQDN_" + who),
		Protocol: os.Getenv("PROTOCOL_" + who),
	}

	return config, nil
}
