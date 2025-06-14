#!/bin/bash

if [ "$1" != "" ]; then
    site="$1"  
else
    site="1"
fi


# --- Configuration ---
MESSAGE="fin"
TARGET_FIFO="/tmp/in_A$site"

# --- Vérification et Envoi ---
echo "Tentative d'envoi du message '$MESSAGE' à la FIFO : $TARGET_FIFO"

# Vérifier si la cible existe et est bien un tube nommé (FIFO)
if [ -p "$TARGET_FIFO" ]; then
    
    # Écrire le message dans la FIFO.
    echo "$MESSAGE" > "$TARGET_FIFO"
    
    echo "Message '$MESSAGE' envoyé avec succès à '$TARGET_FIFO'."
else
    echo "Erreur : La FIFO '$TARGET_FIFO' n'existe pas ou n'est pas un tube nommé."
    exit 1 # Quitter avec un code d'erreur
fi
