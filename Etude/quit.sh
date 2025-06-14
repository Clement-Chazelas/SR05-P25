#!/bin/bash

# Vérifier qu'au moins un argument (le numéro du site parent) est fourni
if [ -z "$1" ]; then
    echo "Utilisation: $0 numero_site_parent [numero_site_enfant...]"
    echo "Exemple: $0 1 6 7"
    echo "Ce script modifie les liens FIFO pour le site parent en y ajoutant les FIFOs des enfants."
    exit 1
fi

# Le premier argument est le numéro du site parent
parent_site=$1
# Supprime le premier argument pour ne garder que les enfants
shift
# Le reste des arguments sont les numéros des sites enfants
children_sites=("$@")

# Chemin vers le fichier contenant les informations du site parent
parent_file="/tmp/${parent_site}.txt"

# Vérifier si le fichier du parent existe
if [ ! -f "$parent_file" ]; then
    echo "Erreur: Fichier de configuration $parent_file non trouvé."
    exit 1
fi

# Lire la première ligne (chaîne des FIFOs) et la deuxième ligne (PID) du fichier du parent
parent_fifo_name=$(head -n 1 "$parent_file")
current_fifo_line=$(head -n 2 "$parent_file" | tail -n 1)
pid_parent=$(tail -n 1 "$parent_file")


# Construire la nouvelle chaîne de FIFOs en ajoutant les FIFOs des enfants
new_parent_fifo_line="$current_fifo_line"
for child in "${children_sites[@]}"; do
    
    child_file="/tmp/${child}.txt"
    
    child_fifo_name=$(head -n 1 "$child_file")
    current_fifo_line_child=$(head -n 2 "$child_file" | tail -n 1)
    pid_child=$(tail -n 1 "$child_file")
    new_fifo_line_child="$current_fifo_line_child /tmp/in_N$parent_fifo_name"
    
    #echo "Nouvelles FIFOs à utiliser pour l'enfant $child: $new_fifo_line_child"
    
    kill -KILL $(pgrep -f /tmp/out_N$child_fifo_name) 
    kill -KILL $pid_child

    cat "/tmp/out_N${child_fifo_name}" | tee $new_fifo_line_child > /dev/null &
    new_pid_child=$! # Capture le nouveau PID du processus 'tee' de l'enfant
    

    # Mettre à jour le fichier de configuration de l'enfant avec les nouvelles informations
    echo "$child_fifo_name" > "$child_file"
    echo "$new_fifo_line_child" >> "$child_file"
    echo "$new_pid_child" >> "$child_file"

    new_parent_fifo_line="$new_parent_fifo_line /tmp/in_N$child_fifo_name"
done

#echo "Nouvelles FIFOs à utiliser pour le parent: $new_parent_fifo_line"

kill -KILL $(pgrep -f /tmp/out_N$parent_fifo_name) 
kill -KILL $pid_parent

# Relancer la commande 'cat | tee' avec les nouvelles FIFOs
cat "/tmp/out_N${parent_fifo_name}" | tee $new_parent_fifo_line > /dev/null &
new_pid=$! # Capture le nouveau PID du processus 'tee'

# Mettre à jour le fichier de configuration du parent avec les nouvelles informations
echo "$parent_fifo_name" > "$parent_file"
echo "$new_fifo_line" >> "$parent_file"
echo "$new_pid" >> "$parent_file"

#echo "fin du script"

