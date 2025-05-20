#!/bin/bash

# Variable pour stocker le flag -v=... à passer aux commandes au lancement de l'application
verbose_flag="-v=false" # Valeur par défaut (verbose désactivé)

# Vérifier si le premier argument est égal à "v"
if [ "$1" = "v" ]; then
    verbose_flag="-v=true" 
    echo "Mode verbeux activé pour App." 
else
    echo "Mode verbeux désactivé pour App." 
fi

if [ ! -e "/tmp/in_A1" ]; then
	mkfifo /tmp/in_A1 /tmp/out_A1
	mkfifo /tmp/in_C1 /tmp/out_C1

	mkfifo /tmp/in_A2 /tmp/out_A2
	mkfifo /tmp/in_C2 /tmp/out_C2

	mkfifo /tmp/in_A3 /tmp/out_A3
	mkfifo /tmp/in_C3 /tmp/out_C3
fi
 
# Fonction de nettoyage déclenchée à l'arrêt du script
nettoyer () {
  echo "Nettoyage des processus et des fifo"
  killall app 2> /dev/null
  killall ctl 2> /dev/null
  killall tee 2> /dev/null
  killall cat 2> /dev/null
  rm -f /tmp/in_* /tmp/out_*
  exit 0
}

# Déclenche le nettoyage sur Ctrl+C (SIGINT) ou sortie
trap nettoyer INT QUIT TERM

./application/app -n A1 "${verbose_flag}" < /tmp/in_A1 > /tmp/out_A1 &
./controleur/ctl -n C1 < /tmp/in_C1 > /tmp/out_C1 &

./application/app -n A2 "${verbose_flag}" < /tmp/in_A2 > /tmp/out_A2 &
./controleur/ctl -n C2 < /tmp/in_C2 > /tmp/out_C2 &

./application/app -n A3 "${verbose_flag}" < /tmp/in_A3 > /tmp/out_A3 &
./controleur/ctl -n C3 < /tmp/in_C3 > /tmp/out_C3 &
 
cat /tmp/out_A1 > /tmp/in_C1 &
cat /tmp/out_C1 | tee /tmp/in_A1 > /tmp/in_C2 &

cat /tmp/out_A2 > /tmp/in_C2 &
cat /tmp/out_C2 | tee /tmp/in_A2 > /tmp/in_C3 &

cat /tmp/out_A3 > /tmp/in_C3 &
cat /tmp/out_C3 | tee /tmp/in_A3 > /tmp/in_C1 &

wait



