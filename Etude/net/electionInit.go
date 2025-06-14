package main

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

const (
	rouge       = "rouge"
	bleu        = "bleu"
	nombreSites = "nbsites"
)

var (
	elu               = math.MaxInt
	parent            = 0
	enfants           = make([]int, 0)
	nbVoisinsAttendus = NbVoisins
	nbDescendants     = 0 // Utilisé pour compter le nombre total de sites
	win               = false
)

func DemarrerElectionInit() {
	if parent == 0 {
		stderr.Println(blanc, "["+Nom+"]", "Je démarre une election", raz)
		elu = MyId
		parent = MyId
		nbVoisinsAttendus = NbVoisins
		envoyerA(electionInit, bleu, strconv.Itoa(elu), ListVoisins)
	}
}

func RecevoirMessageBleuInit(msg string) {
	k, _ := strconv.Atoi(findval(msg, MsgData))
	senderId, _ := strconv.Atoi(findval(msg, MsgSender))
	if k < elu {
		stderr.Println(noir, "["+Nom+"]", "Je change de vague pour", k, raz)
		elu = k
		parent = senderId
		enfants = make([]int, 0)
		nbVoisinsAttendus = NbVoisins - 1
		nbDescendants = 0
		if nbVoisinsAttendus > 0 {
			envoyerAuxVoisinsSauf(electionInit, bleu, strconv.Itoa(elu), senderId)
		} else {
			// j'envoie à mon parent mon nombre de descendants
			envoyerAuParent(electionInit, rouge, strconv.Itoa(elu), nbDescendants)
			stderr.Println(cyan, "["+Nom+"]", "Je n'ai pas d'enfant", raz)
		}
	} else if elu == k {
		destinataire := []int{senderId}
		envoyerA(electionInit, rouge, strconv.Itoa(elu), destinataire)
	}
}

func RecevoirMessageRougeInit(msg string) {
	k, _ := strconv.Atoi(findval(msg, MsgData))
	senderId, _ := strconv.Atoi(findval(msg, MsgSender))
	if elu == k {

		nbVoisinsAttendus--
		if findval(msg, "enfant") != "" {
			// Le message a été envoyé par mon enfant
			enfants = append(enfants, senderId)
			nbRecu, _ := strconv.Atoi(findval(msg, "enfant"))
			nbDescendants += nbRecu + 1
		}
		if nbVoisinsAttendus == 0 {
			if elu == MyId {

				win = true
				NbSites = nbDescendants + 1
				stderr.Println(cyan, "["+Nom+"]", "Mes enfants sont", enfants, raz)
				stderr.Println(orange, "["+Nom+"]", "J'ai gagné, le nombre de site est :", NbSites, raz)
			} else {
				// j'envoie à mon parent mon nombre de descendants
				envoyerAuParent(electionInit, rouge, strconv.Itoa(elu), nbDescendants)
				if len(enfants) != 0 {
					stderr.Println(cyan, "["+Nom+"]", "Mes enfants sont", enfants, raz)
				} else {
					stderr.Println(cyan, "["+Nom+"]", "Je n'ai pas d'enfant", raz)
				}
			}
		}
	}
}

func recevoirMessageElectionInit(msg string) {
	destinataires := findval(msg, MsgDestination)

	if destinataires != "" && !slices.Contains(strToIntTab(destinataires), MyId) {
		// Ce message ne m'était pas déstiné
		return
	}

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

func envoyerAuxVoisinsSauf(msgCat string, couleur string, data string, saufVoisin int) {
	if !slices.Contains(ListVoisins, saufVoisin) {
		stderr.Println(rouge, "["+Nom+"]", "Voisin non présent dans la liste", saufVoisin, ListVoisins, raz)
		return
	}
	vIndex := slices.Index(ListVoisins, saufVoisin)
	tmpSlice := make([]int, len(ListVoisins))
	copy(tmpSlice, ListVoisins)
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, msgCat) + MsgFormat(MsgType, couleur) +
		MsgFormat(MsgData, data) +
		MsgFormat(MsgDestination, intTabToStr(append(tmpSlice[:vIndex], tmpSlice[vIndex+1:]...)))
	fmt.Println(msg)
}

func envoyerAuParent(msgCat string, msgType string, data string, nbDescendants int) {
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, msgCat) + MsgFormat(MsgType, msgType) +
		MsgFormat(MsgData, data) + MsgFormat(MsgDestination, strconv.Itoa(parent)) +
		MsgFormat("enfant", strconv.Itoa(nbDescendants))
	fmt.Println(msg)
}

func envoyerA(msgCat string, msgType string, data string, destinaires []int) {
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, msgCat) + MsgFormat(MsgType, msgType) +
		MsgFormat(MsgData, data) + MsgFormat(MsgDestination, intTabToStr(destinaires))
	fmt.Println(msg)
}

func intTabToStr(voisins []int) string {
	strs := make([]string, len(voisins))
	for i, v := range voisins {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, ",")
}

func strToIntTab(voisinsStr string) []int {
	parts := strings.Split(voisinsStr, ",")
	voisins := make([]int, len(parts))
	for i, val := range parts {
		voisins[i], _ = strconv.Atoi(val)
	}
	return voisins
}
