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

func DemarrerElection() {
	if parent == 0 {
		elu = MyId
		parent = MyId
		envoyerA(bleu, strconv.Itoa(elu), ListVoisins)
	}
}

func RecevoirMessageBleu(msg string) {
	k, _ := strconv.Atoi(findval(msg, MsgData))
	senderId, _ := strconv.Atoi(findval(msg, MsgSender))
	if k < elu {
		elu = k
		parent = senderId
		enfants = make([]int, 0)
		nbVoisinsAttendus = NbVoisins - 1
		if nbVoisinsAttendus > 0 {
			envoyerAuxVoisinsSauf(bleu, strconv.Itoa(elu), senderId)
		} else {
			// j'envoie à mon parent mon nombre de descendants
			envoyerAuParent(rouge, strconv.Itoa(elu), nbDescendants)
		}
	} else if elu == k {
		destinataire := []int{senderId}
		envoyerA(rouge, strconv.Itoa(elu), destinataire)
	}
}

func RecevoirMessageRouge(msg string) {
	k, _ := strconv.Atoi(findval(msg, MsgData))
	senderId, _ := strconv.Atoi(findval(msg, MsgSender))
	if elu == k {
		NbVoisins--
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
			} else {
				// j'envoie à mon parent mon nombre de descendants
				envoyerAuParent(rouge, strconv.Itoa(elu), nbDescendants)
			}

		}
	}
}

func recevoirMessageElection(msg string) {
	destinataires := findval(msg, MsgDestination)
	if destinataires != "" && !slices.Contains(strToIntTab(destinataires), MyId) {
		// Ce message ne m'était pas déstiné
		return
	}

	switch findval(msg, MsgType) {
	case bleu:
		RecevoirMessageBleu(msg)
	case rouge:
		RecevoirMessageRouge(msg)
	case nombreSites:
		NbSites, _ = strconv.Atoi(findval(msg, MsgData))
		// Je transmets à mes enfants le nombre de sites
		envoyerA(nombreSites, strconv.Itoa(NbSites), enfants)
	}
}

func envoyerAuxVoisinsSauf(couleur string, data string, saufVoisin int) {
	if !slices.Contains(ListVoisins, saufVoisin) {
		stderr.Println("Voisin non présent dans la liste")
	}
	vIndex := slices.Index(ListVoisins, saufVoisin)
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, electionInit) + MsgFormat(MsgType, couleur) +
		MsgFormat(MsgData, data) +
		MsgFormat(MsgDestination, intTabToStr(append(ListVoisins[:vIndex], ListVoisins[vIndex+1:]...)))
	fmt.Println(msg)
}

func envoyerAuParent(msgType string, data string, nbDescendants int) {
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, electionInit) + MsgFormat(MsgType, msgType) +
		MsgFormat(MsgData, data) + MsgFormat(MsgDestination, strconv.Itoa(parent)) +
		MsgFormat("enfant", strconv.Itoa(nbDescendants))
	fmt.Println(msg)
}

func envoyerA(msgType string, data string, destinaires []int) {
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, electionInit) + MsgFormat(MsgType, msgType) +
		MsgFormat(MsgData, data) + MsgFormat(MsgDestination, intTabToStr(destinaires))
	fmt.Println(msg)
}

func intTabToStr(voisins []int) string {
	strs := make([]string, len(ListVoisins))
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
