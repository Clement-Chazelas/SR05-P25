# SR05-P25
Projet et devoirs (activités) SR05 semestre P25 | UTC

## Présentation générale

Ce projet consiste en une application répartie simulant le fonctionnement d'une blockchain. 

L'implémentation a été faite en Go et est constituée d'une partie application et d'une partie contrôleur comportant notamment un algorithme de file d'attente répartie et un algorithme de sauvegarde. Les contrôleurs ont été conçus pour communiquer au sein d'un réseau en anneau unidirectionnel, c'est-à-dire que chaque message reçu est relayé au site suivant jusqu'à atteindre l'expéditeur lui-même qui stop la boucle.

## Lancement
L'exécution débute par l'initialisation des contrôleurs. Cette initialisation consiste en l'échange de leur nom afin de déterminer leur identifiant (indice de l'ordre alphabétique). Une fois cette phase terminée, le contrôleur prévient son application pour qu'elle puisse débuter. Chaque application va génèrer sa propre clé publique/privée et va ensuite partager sa clé avec les autres sites, tout en récupérant celles des autres. Une fois l'échange de clés terminé l’application ayant la plus grande clé devient l’initiateur. Elle va créer le premier bloc puis l’envoyer aux autres. 
   
## Partie application
L’application représente un site de la blockchain et gère toute la logique métier liée à la gestion de la chaîne de blocs.
Elle fonctionne en collaboration étroite avec son contrôleur associé pour garantir la cohérence et la sécurité du système réparti.
Elle reprend les bases de l'activité 4, garantissant une exécution séquentielle, des actions de lectures et d'écritures atomiques, et une lecture asynchrone.

Principales responsabilités :

**Initialisation**
- Génération de la paire de clés publique/privée
- Échange des clés publiques et des noms avec les autres sites via le contrôleur
- Création du bloc initial par le site ayant la plus grande clé publique, puis diffusion à tous les autres

**Gestion des transactions**
- Création de transactions (expéditeur, destinataire, montant, timestamp, signature)
- Signature des transactions avec la clé privée locale
- Envoi des transactions aux autres sites via le contrôleur
- Réception, vérification (signature, solde suffisant) et ajout des transactions reçues dans le pool d’attente

**Minage et section critique**
- Demande d’accès à la section critique (SC) auprès du contrôleur pour miner un bloc lorsqu'il existe au moins une transaction en attente
- Regroupement des transactions en attente dans un nouveau bloc
- Calcul du hash du bloc (preuve de travail : hash commençant par 00000)
- Ajout du bloc miné à la blockchain locale
- Envoi du bloc miné aux autres sites via le contrôleur

**Propagation et validation**
- Réception des blocs minés par les autres sites
- Vérification de l’intégrité du bloc (hash, previousHash, validité des transactions, cohérence des UTXO (solde de chaque site))
- Mise à jour de la blockchain locale et du pool de transactions : les transactions déjà minées sont retirées

**Communication**
- Tous les échanges au sein du réseau passent par le contrôleur associé, qui relaie les messages aux autres sites en le formattant (ajout du nom, catégorie du message, horloge vectorielle, couleur du contrôleur).

## Partie contrôleur

Le contrôleur agit comme un médiateur et coordinateur pour l’application.
Il assure la synchronisation, la diffusion fiable des messages, la gestion de l’exclusion mutuelle (minage) et la capture d’instantanés (snapshots)

Principales responsabilités :

**Initialisation et identification**
- Échange des noms entre contrôleurs pour constituer la liste globale des sites
- Attribution d’un identifiant unique à chaque contrôleur (index dans la liste triée)
- Transmission du signal de départ à l’application une fois l’initialisation terminée

**Gestion des messages**
- Lecture continue des messages entrants (depuis l’application ou d’autres contrôleurs
- Filtrage des messages pour éviter les doublons (car anneau unidirectionnel entre les controleurs)
- Relais des messages applicatifs, de file d’attente et de snapshot selon leur catégorie

**Algorithme de file d’attente répartie**
- Réception des demandes d’accès à la section critique de l’application
- Diffusion des requêtes, accusés de réception (ack) et libérations (release) aux autres contrôleurs
- Maintien d’une file d’attente locale des requêtes, triée par estampille et identifiant
- Autorisation de l’accès à la SC uniquement si la requête locale est la plus ancienne
- Transmission du signal d’entrée/sortie de SC à l’application

**Gestion des snapshots distribués**
- Déclenchement et gestion de la capture d’instantanés de l’état global
- Utilisation d’horloges vectorielles pour dater les snapshots
- Agrégation des états locaux et des messages prépost pour obtenir une image cohérente du système

**Sérialisation et conversion**
- Conversion des structures complexes (transactions, blocs, blockchain) en chaînes JSON pour la transmission réseau

## Algorithme d'exécution répartie

Pour garantir la cohérence et l’exclusivité lors du minage, chaque application doit demander l’accès à la section critique (SC) via son contrôleur
Le contrôleur utilise un algorithme de file d’attente répartie pour coordonner l’accès à la SC entre tous les sites

Déroulement :

1. Demande d’accès à la SC :  
L’application envoie FILE:demandeSC à son contrôleur  
2. Propagation de la requête :  
Le contrôleur diffuse une requête à tous les autres contrôleurs, en utilisant une estampille logique (numéro croissant)  
3. File d’attente répartie :  
Chaque contrôleur maintient une file d’attente locale des requêtes reçues, triées par estampille et identifiant  
4. Accès à la SC :  
Un site obtient l’accès à la SC uniquement si sa requête est la plus ancienne (plus petite estampille)  
5. Début de la SC :  
Le contrôleur envoie CONT:debutSC à son application, qui peut alors miner un bloc  
6. Libération de la SC :  
Après le minage, l’application envoie FILE:finSC à son contrôleur, qui va alors diffuser un message de libération à tous les autres  

## Algorithme de sauvegarde

Un algortithme de calcul d'instantané (snapshot) a également été mis en place afin d'avoir une image cohérente de l'état global du système, c'est-à-dire essentiellement de la blockchain et les messages en transit (messages prepost). Il est réalisé dans l'optique de réaliser des sauvegardes ou de reprendre l'état du système en cas de défaillance. 

Ainsi, l'approche utilisée dans snapshot.go s'appuie sur le concept d'horloges vectorielles afin de dater correctement les snapshots.

### Fonctionnement 

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
