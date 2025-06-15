package main

/*
Ce fichier implémente l'algorithme de l'élection par extinction de vague.
Il est utilisé lors de l'ajout ou d'un départ d'un site.
L'arborescence issue de ces vagues n'est pas sauvegardé et n'impact pas l'arborescence initiale.
Les fonctions envoyerXXX() sont définies dans le fichier electionInit.
*/

import (
	"math"
	"slices"
	"strconv"
)

var (
	parentTmp = 0 // Parent de l'arborescence temporaire
)

// DemarrerElection permet de démarrer une élection, si je ne fais pas déjà partie d'une arborescence.
func DemarrerElection() {
	if parentTmp == 0 {
		stderr.Println(blanc, "["+Nom+"]", "Je démarre une election", raz)
		elu = MyId
		parentTmp = MyId
		nbVoisinsAttendus = NbVoisins
		// Envoi d'un message bleu contenant mon ID à mes voisins
		envoyerA(election, bleu, strconv.Itoa(elu), ListVoisins)
	}
}

// RecevoirMessageBleu permet de traiter les messages bleus reçus lors d'une élection
func RecevoirMessageBleu(msg string) {
	// Récupération de l'élu de l'expéditeur et de son ID
	k, _ := strconv.Atoi(findval(msg, MsgData))
	senderId, _ := strconv.Atoi(findval(msg, MsgSender))

	// Si l'ID de son élu est inférieur à l'ID de mon élu, je change de vague
	if k < elu {
		stderr.Println(noir, "["+Nom+"]", "Je change de vague pour", k, raz)
		// Changement d'élu, et l'expéditeur devient mon parent
		elu = k
		parentTmp = senderId
		// Réinitialisation du nombre de voisins
		nbVoisinsAttendus = NbVoisins - 1

		if nbVoisinsAttendus > 0 {
			// J'ai des voisins, je leur envoie un message bleu contenant mon élu
			envoyerAuxVoisinsSauf(election, bleu, strconv.Itoa(elu), senderId)
		} else {
			// Sinon, j'envoie rouge à mon parent
			destinataire := []int{parentTmp}
			envoyerA(election, rouge, strconv.Itoa(elu), destinataire)
		}
	} else if elu == k {
		// Je fais déjà partie de cette vague, j'envoie un message rouge à l'expéditeur
		destinataire := []int{senderId}
		envoyerA(election, rouge, strconv.Itoa(elu), destinataire)
	}
}

// RecevoirMessageRouge permet de traiter les messages rouges reçus lors d'une élection
func RecevoirMessageRouge(msg string) {
	// Récupération de l'élu de l'expéditeur
	k, _ := strconv.Atoi(findval(msg, MsgData))

	// l'élu correspond à ma vague
	if elu == k {
		nbVoisinsAttendus--

		// Je n'attends plus de réponse de mes voisins
		if nbVoisinsAttendus == 0 {
			// Je suis l'élu de la vague
			if elu == MyId {
				// J'ai gagné l'élection
				win = true
				stderr.Println(orange, "["+Nom+"]", "J'ai gagné", raz)
			} else {
				// J'envoie rouge à mon parent
				destinataire := []int{parentTmp}
				envoyerA(election, rouge, strconv.Itoa(elu), destinataire)
			}
		}
	}
}

// recevoirMessageElection traite tous les messages de la catégorie election.
func recevoirMessageElection(msg string) {
	// Récupération de la liste des destinataires du message
	destinataires := findval(msg, MsgDestination)

	// Si je ne suis pas dedans, je ne le traite pas
	if destinataires != "" && !slices.Contains(strToIntTab(destinataires), MyId) {
		return
	}

	// Traitement du message en fonction de son type
	switch findval(msg, MsgType) {
	case bleu:
		RecevoirMessageBleu(msg)
		break
	case rouge:
		RecevoirMessageRouge(msg)
		break
	}
}

// resetElection permet de mettre à zero les variables de vagues pour permettre une nouvelle élection
func resetElection() {
	elu = math.MaxInt
	nbVoisinsAttendus = NbVoisins
	parentTmp = 0
	win = false
}
