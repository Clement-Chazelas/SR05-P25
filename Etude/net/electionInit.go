package main

/*
Ce fichier implémente l'algorithme de l'élection par extinction de vague.
Il est utilisé lors de l'initialisation des NET pour mettre en place une arborescence et compter le nombre de sites.
*/

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// Type de message d'une élection
const (
	rouge       = "rouge"
	bleu        = "bleu"
	nombreSites = "nbsites"
)

// Variable nécessaire à l'élection
var (
	elu               = math.MaxInt
	parent            = 0
	enfants           = make([]int, 0)
	nbVoisinsAttendus = NbVoisins
	nbDescendants     = 0 // Utilisé pour compter le nombre total de sites
	win               = false
)

// DemarrerElectionInit permet de démarrer une élection, si je ne fais pas déjà partie d'une arborescence.
func DemarrerElectionInit() {
	// Je n'ai pas de parent défini
	if parent == 0 {
		stderr.Println(blanc, "["+Nom+"]", "Je démarre une election", raz)
		elu = MyId
		parent = MyId
		nbVoisinsAttendus = NbVoisins
		// Envoi d'un message bleu contenant mon ID à mes voisins
		envoyerA(electionInit, bleu, strconv.Itoa(elu), ListVoisins)
	}
}

// RecevoirMessageBleuInit permet de traiter les messages bleus reçus lors d'une élection
func RecevoirMessageBleuInit(msg string) {
	// Récupération de l'élu de l'expéditeur et de son ID
	k, _ := strconv.Atoi(findval(msg, MsgData))
	senderId, _ := strconv.Atoi(findval(msg, MsgSender))

	// Si l'ID de son élu est inférieur à l'ID de mon élu, je change de vague
	if k < elu {
		stderr.Println(noir, "["+Nom+"]", "Je change de vague pour", k, raz)
		// Changement d'élu, et l'expéditeur devient mon parent
		elu = k
		parent = senderId
		// Réinitialisation des variables pour la nouvelle vague
		enfants = make([]int, 0)
		nbVoisinsAttendus = NbVoisins - 1
		nbDescendants = 0

		if nbVoisinsAttendus > 0 {
			// J'ai des voisins, je leur envoie un message bleu contenant mon élu
			envoyerAuxVoisinsSauf(electionInit, bleu, strconv.Itoa(elu), senderId)
		} else {
			// Sinon, j'envoie à mon parent mon nombre de descendants (0)
			envoyerAuParent(electionInit, rouge, strconv.Itoa(elu), nbDescendants)
			stderr.Println(cyan, "["+Nom+"]", "Je n'ai pas d'enfant", raz)
		}
	} else if elu == k {
		// Je fais déjà partie de cette vague, j'envoie un message rouge à l'expéditeur
		destinataire := []int{senderId}
		envoyerA(electionInit, rouge, strconv.Itoa(elu), destinataire)
	}
}

// RecevoirMessageRougeInit permet de traiter les messages rouges reçus lors d'une élection
func RecevoirMessageRougeInit(msg string) {
	// Récupération de l'élu de l'expéditeur et de son ID
	k, _ := strconv.Atoi(findval(msg, MsgData))
	senderId, _ := strconv.Atoi(findval(msg, MsgSender))

	// l'élu correspond à ma vague
	if elu == k {
		nbVoisinsAttendus--

		// Le message a été envoyé par mon enfant
		if findval(msg, "enfant") != "" {
			// Je l'ajoute à ma liste d'enfant
			enfants = append(enfants, senderId)
			// Je récupère son nombre de descendants et l'ajoute au mien
			nbRecu, _ := strconv.Atoi(findval(msg, "enfant"))
			nbDescendants += nbRecu + 1
		}

		// Je n'attends plus de réponse de mes voisins
		if nbVoisinsAttendus == 0 {
			// Je suis l'élu de la vague
			if elu == MyId {
				// J'ai gagné l'élection
				win = true
				// Le nombre de sites total est mon nombre de descendants + 1 (moi)
				NbSites = nbDescendants + 1

				stderr.Println(cyan, "["+Nom+"]", "Mes enfants sont", enfants, raz)
				stderr.Println(orange, "["+Nom+"]", "J'ai gagné, le nombre de site est :", NbSites, raz)
			} else {
				// j'envoie à mon parent mon nombre de descendants
				envoyerAuParent(electionInit, rouge, strconv.Itoa(elu), nbDescendants)
				// Uniquement pour l'affichage
				if len(enfants) != 0 {
					stderr.Println(cyan, "["+Nom+"]", "Mes enfants sont", enfants, raz)
				} else {
					stderr.Println(cyan, "["+Nom+"]", "Je n'ai pas d'enfant", raz)
				}
			}
		}
	}
}

// recevoirMessageElectionInit traite tous les messages de la catégorie electionInit.
func recevoirMessageElectionInit(msg string) {

	// Récupération de la liste des destinataires du message
	destinataires := findval(msg, MsgDestination)

	// Si je ne suis pas dedans, je ne le traite pas
	if destinataires != "" && !slices.Contains(strToIntTab(destinataires), MyId) {
		// Ce message ne m'était pas déstiné
		return
	}

	// Traitement du message en fonction de son type
	switch findval(msg, MsgType) {
	case bleu:
		RecevoirMessageBleuInit(msg)
	case rouge:
		RecevoirMessageRougeInit(msg)
	case nombreSites:
		NbSites, _ = strconv.Atoi(findval(msg, MsgData))
		// Je transmets à mes enfants le nombre de sites
		envoyerA(electionInit, nombreSites, strconv.Itoa(NbSites), enfants)
	}
}

// envoyerAuxVoisinsSauf permet d'envoyer un message de catégorie election à tous ses voisins sauf un seul.
func envoyerAuxVoisinsSauf(msgCat string, couleur string, data string, saufVoisin int) {
	// Vérification que le voisin est présent dans la liste des voisins
	if !slices.Contains(ListVoisins, saufVoisin) {
		stderr.Println(rouge, "["+Nom+"]", "Voisin non présent dans la liste", saufVoisin, ListVoisins, raz)
		return
	}
	// Récupération de l'indice du voisin à éviter dans ma liste
	vIndex := slices.Index(ListVoisins, saufVoisin)
	// Utilisation d'un tableau temporaire pour éviter les modifications de pointeur
	tmpSlice := make([]int, len(ListVoisins))
	copy(tmpSlice, ListVoisins)

	// Envoi du message avec comme destinataire tous mes voisins sauf celui indiqué
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, msgCat) + MsgFormat(MsgType, couleur) +
		MsgFormat(MsgData, data) +
		MsgFormat(MsgDestination, intTabToStr(append(tmpSlice[:vIndex], tmpSlice[vIndex+1:]...)))
	fmt.Println(msg)
}

// envoyerAuParent permet d'envoyer un message de catégorie election à son parent.
func envoyerAuParent(msgCat string, msgType string, data string, nbDescendants int) {
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, msgCat) + MsgFormat(MsgType, msgType) +
		MsgFormat(MsgData, data) + MsgFormat(MsgDestination, strconv.Itoa(parent)) +
		MsgFormat("enfant", strconv.Itoa(nbDescendants))
	fmt.Println(msg)
}

// envoyerA permet d'envoyer un message de catégorie election à un site en particulier.
func envoyerA(msgCat string, msgType string, data string, destinaires []int) {
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, msgCat) + MsgFormat(MsgType, msgType) +
		MsgFormat(MsgData, data) + MsgFormat(MsgDestination, intTabToStr(destinaires))
	fmt.Println(msg)
}

// intTabToStr convertit un tableau d'entiers en une chaine de caractères pour l'envoi de message
func intTabToStr(tab []int) string {
	strs := make([]string, len(tab))
	for i, v := range tab {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, ",")
}

// intTabToStr convertit une chaine de caractères en tableau d'entiers pour la lecture de message
func strToIntTab(tabStr string) []int {
	parts := strings.Split(tabStr, ",")
	voisins := make([]int, len(parts))
	for i, val := range parts {
		voisins[i], _ = strconv.Atoi(val)
	}
	return voisins
}
