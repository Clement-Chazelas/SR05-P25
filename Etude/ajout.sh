#!/bin/bash

if [ -z "$1" ]; then
    echo "Utilisation: $0 id_site_parent id_nouvea_site"
    echo "Exemple: $0 1 6 "
    exit 1
fi

parent_id=$1
nouveau_id=$2

mkfifo /tmp/in_A$nouveau_id /tmp/out_A$nouveau_id
mkfifo /tmp/in_C$nouveau_id /tmp/out_C$nouveau_id
mkfifo /tmp/in_N$nouveau_id /tmp/out_N$nouveau_id


for file in /tmp/*.txt; do
        # Lit la première ligne du fichier
	first_line=$(head -n 1 "$file")

        # Compare la première ligne avec le texte recherché
        if [ "$first_line" = "$parent_id" ]; then
        	current_fifo_line=$(head -n 2 "$file" | tail -n 1)
		pid_parent=$(tail -n 1 "$file")
		parent_file=$file
            break # S'arrête dès que le fichier est trouvé
        fi
done

new_fifo_line="$current_fifo_line /tmp/in_N$nouveau_id"
kill -KILL $(pgrep -f /tmp/out_N$parent_id) 
kill -KILL $pid_parent

cat /tmp/out_N$parent_id | tee $new_fifo_line > /dev/null &
nv_pid_parent=$!
cat /tmp/out_N$nouveau_id | tee /tmp/in_C$nouveau_id /tmp/in_N$parent_id > /dev/null&
pid_nv=$!
cat /tmp/out_C$nouveau_id | tee /tmp/in_A$nouveau_id /tmp/in_N$nouveau_id > /dev/null &
cat /tmp/out_A$nouveau_id > /tmp/in_C$nouveau_id &  


./application/app -n "A$nouveau_id" -new=true < /tmp/in_A$nouveau_id > /tmp/out_A$nouveau_id 2>> /tmp/affichage.log &
./controleur/ctl -n "C$nouveau_id" -new=true < /tmp/in_C$nouveau_id > /tmp/out_C$nouveau_id 2>> /tmp/affichage.log &
./net/nett -n "N$nouveau_id" -new=true < /tmp/in_N$nouveau_id > /tmp/out_N$nouveau_id 2>> /tmp/affichage.log &
pid_net=$!

echo "$parent_id" > "$parent_file"
echo "$new_fifo_line" >> "$parent_file"
echo "$nv_pid_parent" >> "$parent_file"

echo "$nouveau_id" > /tmp/$pid_net.txt
echo "/tmp/in_C$nouveau_id /tmp/in_N$parent_id" >> /tmp/$pid_net.txt
echo "$pid_nv" >> /tmp/$pid_net.txt
