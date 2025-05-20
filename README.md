# SR05-P25
Projet et devoirs (activités) SR05 semestre P25 | UTC

## Présentation générale

Ce projet consiste en une application répartie simulant le fonctionnement d'une blockchain. 

L'implémentation a été faite en Go et est constituée d'une partie application et d'une partie contrôleur comportant notamment un algorithme de file d'attente répartie et un algorithme de sauvegarde.

## Lancement

## Partie application


## Partie contrôleur

### Algorithme d'exécution répartie

Afin de mettre en oeuvre l'exécution répartie, un algortithme tel que vu dans le cours a été mis en place.

### Algorithme de file d'attente répartie

Etant donné qu'un seul site peut miner à la fois pour éviter les conflits et assurer la cohérence de la chaîne, un algortihme de file d'attente répartie a été implémenté. Il permet un accès exclusif à la ressource de minage grâce à la demande d'une section critique. 

### Algorithme de sauvegarde

Un algortithme de calcul d'instantané (snapshot) a également été mis en place afin d'avoir une image cohérente de l'état global du système, c'est-à-dire essentiellement de la blockchain et les messages en transit (messages prepost). Il est réalisé dans l'optique de réaliser des sauvegardes ou de reprendre l'état du système en cas de défaillance. 

Ainsi, l'approche utilisée dans snapshot.go s'appuie sur le concept d'horloges vectorielles afin de dater correctement les snapshots.

#### Fonctionnement 

1. Initialisation : Une variable couleur permettant d'indentifier les sites marqués est initialisée à blanc ainsi qu'une variable initiateur permettant d'indentifier l'initiateur est initialisée à faux. Une variable pour enregistrer l'état global est également prévue.
2. Début de l'instantané : Un des sites initie la capture d'instantané. Il enregistre alors son état local, passe à la couleur rouge et s'indique comme étant l'initiateur. 
3. Réception d'un message applicatif :
   Si un site reçoit un message applicatif rouge alors que lui-même est blanc, il passe sa couleur à rouge et enregistre son état local.
   Sinon, si la couleur de l'expéditeur est blanc et que lui-même est rouge (message envoyé avant que l'expéditeur n'ait pris son instantané), il considère que c'est un message prepost et le renvoie sur l'anneau.
4. Réception d'un message prepost :
   Si le site est l'initiateur, il ajoute le message à sa liste de messages prepost.
   Sinon, il le renvoie sur l'anneau. 
6. Réception d'un message état :
   Si le site est l'initiateur, il enregistre l'état local reçu.
   Sinon, il le renvoie sur l'anneau. 

#### Implémentation

InitSnapshot() : initialise la capture d'instantané sur le site actuel
sendSnapshotMessage() : envoie un message de snapshot formaté
ReceiveAppMessage() : gère les messages applicatifs et détecte les messages prepost pour sauvegarder l'état des canaux
ReceivePrepostMessage() : gère les messages prepost, c'est-à-dire la réception des messages prepost des sites non marqués
ReceiveStateMessage() : gère les messages état, c'est-à-dire la réception d'un état local distant

Des fonctions utilitaires notamment de conversion des types sont également implémentées. 
