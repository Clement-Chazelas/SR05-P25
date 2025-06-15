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
	mkfifo /tmp/in_N1 /tmp/out_N1

	mkfifo /tmp/in_A2 /tmp/out_A2
	mkfifo /tmp/in_C2 /tmp/out_C2
	mkfifo /tmp/in_N2 /tmp/out_N2

	mkfifo /tmp/in_A3 /tmp/out_A3
	mkfifo /tmp/in_C3 /tmp/out_C3
	mkfifo /tmp/in_N3 /tmp/out_N3
	
	mkfifo /tmp/in_A4 /tmp/out_A4
	mkfifo /tmp/in_C4 /tmp/out_C4
	mkfifo /tmp/in_N4 /tmp/out_N4
	
	mkfifo /tmp/in_A5 /tmp/out_A5
	mkfifo /tmp/in_C5 /tmp/out_C5
	mkfifo /tmp/in_N5 /tmp/out_N5
	
	touch /tmp/affichage.log
	
fi
 
# Fonction de nettoyage déclenchée à l'arrêt du script
nettoyer () {
  echo "Nettoyage des processus et des fifo"
  killall app 2> /dev/null
  killall ctl 2> /dev/null
  killall nett 2> /dev/null
  killall tee 2> /dev/null
  killall cat 2> /dev/null
  rm -f /tmp/in_* /tmp/out_*
  rm -f /tmp/*.txt
  rm -f /tmp/affichage.log
  exit 0
}

# Déclenche le nettoyage sur Ctrl+C (SIGINT) ou sortie
trap nettoyer INT QUIT TERM

./application/app -n A1 "${verbose_flag}" < /tmp/in_A1 > /tmp/out_A1 2>> /tmp/affichage.log &
./controleur/ctl -n C1 < /tmp/in_C1 > /tmp/out_C1 2>> /tmp/affichage.log &
./net/nett -n N1 < /tmp/in_N1 > /tmp/out_N1 2>> /tmp/affichage.log &
PID_N1=$!

./application/app -n A2 "${verbose_flag}" < /tmp/in_A2 > /tmp/out_A2 2>> /tmp/affichage.log &
./controleur/ctl -n C2 < /tmp/in_C2 > /tmp/out_C2 2>> /tmp/affichage.log &
./net/nett -n N2 < /tmp/in_N2 > /tmp/out_N2 2>> /tmp/affichage.log &
PID_N2=$!

./application/app -n A3 "${verbose_flag}" < /tmp/in_A3 > /tmp/out_A3 2>> /tmp/affichage.log &
./controleur/ctl -n C3 < /tmp/in_C3 > /tmp/out_C3 2>> /tmp/affichage.log &
./net/nett -n N3 < /tmp/in_N3 > /tmp/out_N3 2>> /tmp/affichage.log &
PID_N3=$!

./application/app -n A4 "${verbose_flag}" < /tmp/in_A4 > /tmp/out_A4 2>> /tmp/affichage.log &
./controleur/ctl -n C4 < /tmp/in_C4 > /tmp/out_C4 2>> /tmp/affichage.log &
./net/nett -n N4 < /tmp/in_N4 > /tmp/out_N4 2>> /tmp/affichage.log &
PID_N4=$!

./application/app -n A5 "${verbose_flag}" < /tmp/in_A5 > /tmp/out_A5 2>> /tmp/affichage.log &
./controleur/ctl -n C5 < /tmp/in_C5 > /tmp/out_C5 2>> /tmp/affichage.log &
./net/nett -n N5 < /tmp/in_N5 > /tmp/out_N5 2>> /tmp/affichage.log &
PID_N5=$!
 
# Lancement des processus de copies des messages entre fifo
# Pour chaque site, le numéro de la fifo, les sites concernés par le tee et le PID du tee 
# sont stockés dans un fichier $PID_NET.txt 
cat /tmp/out_A1 > /tmp/in_C1 &
cat /tmp/out_C1 | tee /tmp/in_A1 /tmp/in_N1 > /dev/null &
cat /tmp/out_N1 | tee /tmp/in_C1 /tmp/in_N2 /tmp/in_N4 > /dev/null &
PID_1=$!
echo "1" > /tmp/$PID_N1.txt
echo "/tmp/in_C1 /tmp/in_N2 /tmp/in_N4" >> /tmp/$PID_N1.txt
echo "$PID_1" >> /tmp/$PID_N1.txt

cat /tmp/out_A2 > /tmp/in_C2 &
cat /tmp/out_C2 | tee /tmp/in_A2 /tmp/in_N2 > /dev/null &
cat /tmp/out_N2 | tee /tmp/in_C2 /tmp/in_N3 /tmp/in_N1 /tmp/in_N5 > /dev/null &
PID_2=$!
echo "2" > /tmp/$PID_N2.txt
echo "/tmp/in_C2 /tmp/in_N3 /tmp/in_N1 /tmp/in_N5" >> /tmp/$PID_N2.txt
echo "$PID_2" >> /tmp/$PID_N2.txt

cat /tmp/out_A3 > /tmp/in_C3 &
cat /tmp/out_C3 | tee /tmp/in_A3 /tmp/in_N3 > /dev/null &
cat /tmp/out_N3 | tee /tmp/in_C3 /tmp/in_N2 /tmp/in_N5  > /dev/null &
PID_3=$!
echo "3" > /tmp/$PID_N3.txt
echo "/tmp/in_C3 /tmp/in_N2 /tmp/in_N5" >> /tmp/$PID_N3.txt
echo "$PID_3" >> /tmp/$PID_N3.txt

cat /tmp/out_A4 > /tmp/in_C4 &
cat /tmp/out_C4 | tee /tmp/in_A4 /tmp/in_N4 > /dev/null &
cat /tmp/out_N4 | tee /tmp/in_C4 /tmp/in_N1 > /dev/null &
PID_4=$!
echo "4" > /tmp/$PID_N4.txt
echo "/tmp/in_C4 /tmp/in_N1" >> /tmp/$PID_N4.txt
echo "$PID_4" >> /tmp/$PID_N4.txt

cat /tmp/out_A5 > /tmp/in_C5 &
cat /tmp/out_C5 | tee /tmp/in_A5 /tmp/in_N5 > /dev/null &
cat /tmp/out_N5 | tee /tmp/in_C5 /tmp/in_N2 /tmp/in_N3 > /dev/null &
PID_5=$!
echo "5" > /tmp/$PID_N5.txt
echo "/tmp/in_C5 /tmp/in_N2 /tmp/in_N3" >> /tmp/$PID_N5.txt
echo "$PID_5" >> /tmp/$PID_N5.txt

tail -f /tmp/affichage.log

wait



