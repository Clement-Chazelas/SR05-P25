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

Le corps de l'application se trouve dans le fichier `app.go`. Les fichiers `blockchainStruct.go` et `serializeStruc.go` implémentent les différentes structures et fonctions nécessaires au concept de blockchain (block, transaction, UTXO,...) ainsi que des méthodes de conversion en chaine de caractères de ces dernières pour pouvoir les envoyer dans des messages. 

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

**Sérialisation et conversion**
- Conversion des structures complexes (transactions, blocs, blockchain) en chaînes JSON pour la transmission réseau

**Communication**
- Tous les échanges au sein du réseau passent par le contrôleur associé, qui relaie les messages aux autres sites en le formattant (ajout du nom, catégorie du message, horloge vectorielle, couleur du contrôleur).

## Partie contrôleur

Le contrôleur agit comme un médiateur et coordinateur pour l’application.
Il assure la synchronisation, la diffusion fiable des messages, la gestion de l’exclusion mutuelle (minage) et la capture d’instantanés (snapshots)

Le corps du contrôleur se trouve au sein du fichier `controle.go`. Les fichiers `fileAttente.go` et `snapshot.go` implémentent respectivement les fonctions liés à l'algorithme de la file d'attente et celui de la capture d'instantané. Pour simplifier au maximum la compréhension du code, nous avons fait en sorte que nos programme soit au plus proche de la forme de ces deux algorithmes, en reprenant une fonction par garde. Nous avons dû néamoins ajouter quelques fonctions "utilitaires" nécessaire à l'implémentation en go.

Les fichiers `blockchainStruct.go` et `serializeStruct.go` présents dans le dossier `controleur`, sont des copies allégées des fichiers présents dans le dossier `application`, pour des raisons de compatibilité des structures. 

Principales responsabilités :

**Initialisation et identification**
- Échange des noms entre contrôleurs pour constituer la liste globale des sites
- Attribution d’un identifiant unique à chaque contrôleur (index dans la liste triée par ordre alphabétique)
- Transmission du signal de départ à l’application une fois son initialisation terminée

**Gestion des messages**
- Lecture continue des messages entrants (depuis l’application ou d’autres contrôleurs)
- Filtrage des messages pour éviter les messages venant de soi-même ou qui étaient déstinés à une application (car anneau unidirectionnel entre les controleurs)
- Traitement des messages applicatifs, de file d’attente et de snapshot selon leur catégorie

**Algorithme de file d’attente répartie**
- Réception des demandes d’accès à la section critique de l’application
- Diffusion des requêtes, accusés de réception (ack) et libérations (release) aux autres contrôleurs
- Maintien d’une file d’attente locale, contenant les derniers messages de chaque site avec leur estampille.
- Autorisation de l’accès à la SC uniquement si la requête locale est la plus ancienne
- Transmission du signal d’entrée/sortie de SC à l’application

**Gestion des snapshots distribués**
- Déclenchement et gestion de la capture d’instantanés de l’état global
- Utilisation d’horloges vectorielles pour dater les snapshots
- Agrégation des états locaux et des messages prépost pour obtenir une image cohérente du système

## Algorithme d'exécution répartie

Pour garantir la cohérence et l’exclusivité lors du minage, chaque application doit demander l’accès à la section critique (SC) via son contrôleur.
Le contrôleur utilise un algorithme de file d’attente répartie pour coordonner l’accès à la SC entre tous les sites.

Déroulement :

1. Demande d’accès à la SC :  
L’application envoie FILE:demandeSC à son contrôleur.
2. Propagation de la requête :  
Le contrôleur diffuse une requête à tous les autres contrôleurs, en utilisant une estampille logique (numéro croissant).
3. File d’attente répartie :  
Chaque contrôleur maintient une file d’attente locale contenant les derniers messages de chaque site avec leur estampille.  
4. Accès à la SC :  
Un site obtient l’accès à la SC uniquement si sa requête est la plus ancienne (plus petite estampille). En cas d'égalité, la priorité est au site avec le plus petit identifiant.  
5. Début de la SC :  
Le contrôleur envoie CONT:debutSC à son application, qui peut alors miner un bloc.  
6. Libération de la SC :  
Après le minage, l’application envoie FILE:finSC à son contrôleur, qui va alors diffuser un message de libération à tous les autres. 

## Algorithme de sauvegarde

Un algortithme de calcul d'instantané (snapshot) a également été mis en place afin d'obtenir une image cohérente de l'état global du système, c'est-à-dire une capture de la blockchain et des messages en transit (messages prepost). L'intérêt d'un tel algortihme est de réaliser des sauvegardes ou de reprendre l'état du système en cas de défaillance. 

Ainsi, l'approche utilisée dans snapshot.go s'appuie sur le concept d'horloges vectorielles afin de dater correctement les snapshots.

### Fonctionnement 

1. Initialisation : Une variable couleur permettant d'identifier les sites marqués est initialisée à blanc ainsi qu'un booléen permettant d'identifier l'initiateur est initialisée à faux. Une variable pour enregistrer l'état global est également prévue.
2. Début de l'instantané : Un des sites initie la capture d'instantané. Il enregistre alors son état local, passe à la couleur rouge et s'indique comme étant l'initiateur. 
3. Réception d'un message applicatif :
   Si un site reçoit un message applicatif rouge alors que lui-même est blanc, il passe sa couleur à rouge et enregistre son état local. Il envoie son état local dans un message de type "état".
   Sinon, si la couleur de l'expéditeur est blanc et que lui-même est rouge (message envoyé avant que l'expéditeur n'ait pris son instantané), il envoie un message de type prépost contenant le message reçu, pour qu'il soit sauvegardé par l'initiateur.
4. Réception d'un message prepost :
   Si le site est l'initiateur, il ajoute le message à sa liste de messages prepost.
   Sinon, il le renvoie sur l'anneau. 
6. Réception d'un message état :
   Si le site est l'initiateur, il enregistre l'état local reçu.
   Sinon, il le renvoie sur l'anneau. 
7. Fin : Une fois que l'initiateur a reçu tous les états locaux, il enregistre l'état global dans un fichier texte : "sauvegarde.txt".

## Utilisation de l'application

Pour lancer l'application, utilisez le script `launch.sh` depuis votre terminal. Le projet a été testé sur l'os Kali basé sur Debian, il est possible que l'affichage des couleurs dans le terminal diffère selon l'os.

#### Lancement

```bash
./launch.sh [v]
```
Le paramètre v est optionnel et permet d'activer le mode verbeux, dans ce cas les applications génèrent un affichage pour chaque message reçu.

#### Initialisation d'une sauvegarde

Pour démarrer une sauvegarde, utilisez le script `snapshot.sh` depuis votre terminal.

```bash
./snapshot.sh
```

#### Build

Le projet contient les exécutables pré-compilés, mais il est possible de les générer soi-même.

Au sein du dossier `application`
```bash
go get app
go build app
```

Au sein du dossier `controleur`
```bash
go build ctl
```
