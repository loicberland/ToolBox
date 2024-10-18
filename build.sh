#!/bin/bash

# Fonction pour afficher l'aide
function show_help() {
    echo "Usage: ./build.sh [option]"
    echo "Options:"
    echo "  all       Build all"
    echo "  lmba      Build only lmba"
    echo "  perso      Build only perso"
    echo "  api      Build only api"
    echo "  help      Display this help message"
}

# Si aucun argument n'est fourni, afficher l'aide
if [ $# -eq 0 ]; then
    show_help
    exit 1
fi

# Lire les options
case $1 in
    all)
        # Compiler les deux répertoires
        echo "Building lmba-toolbox.exe..."
        go build -o ./_build/lmba-toolbox.exe ./lmba
        echo "lmba-toolbox built successfully."

        echo "Building perso-toolbox.exe..."
        go build -o ./_build/perso-toolbox.exe ./perso
        echo "perso-toolbox.exe built successfully."

        echo "Building api-toolbox.exe..."
        CGO_ENABLED=1 go build -o ./_build/api-toolbox.exe ./api
        echo "api-toolbox.exe built successfully."
        ;;
    lmba)
        # Compiler uniquement lmba-toolbox
        echo "Building lmba-toolbox.exe..."
        go build -o ./_build/lmba-toolbox.exe ./lmba
        echo "lmba-toolbox built successfully."
        ;;
    perso)
        # Compiler uniquement perso
        echo "Building perso-toolbox.exe..."
        go build -o ./_build/perso-toolbox.exe ./perso
        echo "perso-toolbox.exe built successfully."
        ;;
    api)
        # Compiler uniquement api
        echo "Building api-toolbox.exe..."
        CGO_ENABLED=1 go build -o ./_build/api-toolbox.exe ./api
        echo "api-toolbox.exe built successfully."
        ;;
    help)
        # Afficher l'aide
        show_help
        ;;
    *)
        # Option non reconnue, afficher l'aide
        echo "Invalid option: $1"
        show_help
        exit 1
        ;;
esac