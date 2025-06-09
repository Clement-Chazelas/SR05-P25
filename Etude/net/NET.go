package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

var fieldsep = "?"
var keyvalsep = "@"

var red string = "\033[1;31m"
var orange string = "\033[1;33m"
var vert string = "\033[1;32m"
var raz string = "\033[0;00m"

const (
	MsgSender      = "sdr"
	MsgDestination = "dest"
	MsgCategory    = "cat"
	MsgData        = "dat"
	MsgPath        = "pth"
	MsgType        = "typ"
	electionInit   = "eli"
	election       = "elec"
	controleur     = "ctr"
	admission      = "adm"
	admResponse    = "res"
)

var (
	pNom            = flag.String("n", "NET", "Nom du noeud")
	pNouveauSite    = flag.Bool("new", false, "Nouveau site")
	Nom             string
	pid             = os.Getpid()
	stderr          = log.New(os.Stderr, "", 0)
	MyId            = pid
	NbVoisins       int
	NbSites         = -1
	ListVoisins     []int
	blockchainData  = ""
	queueData       = ""
	controllerNames string
	ListEnfants     []int
	waitingSite     int
)

func MsgFormat(key string, val string) string {
	return fieldsep + keyvalsep + key + keyvalsep + val
}

func findval(msg string, key string) string {

	if len(msg) < 4 {
		return ""
	}

	tab_allkeyvals := strings.Split(msg[1:], fieldsep)
	for _, keyval := range tab_allkeyvals {
		tabkeyval := strings.Split(keyval[1:], keyvalsep)
		if tabkeyval[0] == key {
			return tabkeyval[1]
		}
	}
	return ""

}

func demanderAdmission() {
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
		MsgFormat(MsgCategory, admission)
	fmt.Println(msg)

	var received string

	for {
		fmt.Scanln(&received)
		if findval(received, MsgCategory) == admResponse {
			// J'ajoute mon voisin comme parent
			parent, _ = strconv.Atoi(findval(received, MsgSender))

			//On récupère les données de la part du parent
			blockchainData = findval(received, "blockchain")
			NbSites, _ = strconv.Atoi(findval(received, nombreSites))
			queueData = findval(received, "queue")
			controllerNames = findval(received, "controllerNames")

			//On dit au controller de lancer son initialisation
			fmt.Printf("NET:start:%d\n", NbSites)

			//On envoie les données nécessaires à l'initialisation du contrôleur
			fmt.Printf("NET:controleur:%s\n", controllerNames)
			fmt.Printf("NET:queue:%s\n", queueData)
			fmt.Printf("NET:blockchain:%s\n", blockchainData)

			break
		}
	}
}

func finaliserAdmission(senderID int) {

	//On ajoute le demandeur au parent en tant qu'enfant
	ListEnfants = append(ListEnfants, senderID)
	ListVoisins = append(ListVoisins, senderID)
	NbSites++
	NbVoisins++
	//On envoie au nouveau site les infos nécessaires à son initialisation
	infos := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
		MsgFormat(MsgCategory, admResponse) +
		MsgFormat("blockchain", blockchainData) +
		MsgFormat("queue", queueData) +
		MsgFormat("controllerNames", controllerNames)
	MsgFormat(nombreSites, strconv.Itoa(NbSites))
	fmt.Println(infos)

	// Réinitialisation des variables pour la prochaine élection
	resetElection()

}

func initialisation() {
	stderr.Println(Nom, "Initialisation")
	fmt.Println(MyId)

	// Lecture des autres NET pour initialisation
	var received string
	heureDebut := time.Now()
	for time.Since(heureDebut) < 2*time.Second {
		fmt.Scanln(&received)
		idVoisin, err := strconv.Atoi(received)
		if err != nil {
			continue
		}
		stderr.Println(Nom, "recu", received)
		ListVoisins = append(ListVoisins, idVoisin)
		received = ""
	}
	NbVoisins = len(ListVoisins)
	stderr.Println(Nom, "Fin des voisins", ListVoisins)
	DemarrerElectionInit()

	// Tant que je n'ai pas gagné l'élection (je n'ai pas fini de compter le nb de sites)
	for {
		fmt.Scanln(&received)
		msgCat := findval(received, MsgCategory)
		switch msgCat {
		case electionInit:
			recevoirMessageElectionInit(received)
			break
		}
		if NbSites != -1 {
			break
		}
	}

	if win {
		// J'ai remporté l'élection, je transmets le nb de site à mes enfants
		envoyerA(electionInit, nombreSites, strconv.Itoa(NbSites), enfants)
	}
	stderr.Println(Nom, "Fin de l'initialisation")
	fmt.Printf("NET:start:%d\n", NbSites)

	// On réinitialise les variables (hors parent/enfants) pour la prochaine election (ajout/départ)
	resetElection()
}

func majHistorique(msg string) string {
	hist := strToIntTab(findval(msg, MsgPath))
	hist[0] = hist[1]
	hist[1] = MyId
	newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, findval(msg, MsgCategory)) +
		MsgFormat(MsgData, findval(msg, MsgData)) + MsgFormat(MsgPath, intTabToStr(hist))
	return newMsg
}

func main() {
	flag.Parse()
	Nom = *pNom + "-" + strconv.Itoa(pid)

	//On check si c'est un nouveau site ou pas, si oui on demande l'admission, si non alors on s'initialise normalement
	if *pNouveauSite {
		demanderAdmission()
	} else {
		initialisation()
	}

	var rcvmsg string
	for {
		fmt.Scanln(&rcvmsg)
		if rcvmsg[:5] == "CONT:" || rcvmsg[:4] == "NET:" {
			//Ce message n'était pas à destination du NET
			rcvmsg = ""
			continue
		}

		// Traitement des messages
		rcvCat := findval(rcvmsg, MsgCategory)
		if rcvCat != "" {
			// Le message vient d'un autre NET
			msgSdr := findval(rcvmsg, MsgSender)
			sdrId, _ := strconv.Atoi(msgSdr)

			if sdrId != parent && !slices.Contains(enfants, sdrId) {
				// Si le message ne vient pas de mon parent ou de mes enfants,
				//et que ce n'est pas une demande d'admission ou une election, je le rejette
				switch rcvCat {
				case admission:
					DemarrerElection()
					waitingSite = sdrId
					break
				case election:
					recevoirMessageElection(rcvmsg)
					if win {
						finaliserAdmission(waitingSite)
					}
					break
				}
				rcvmsg = ""
				continue
			}

			msgHist := findval(rcvmsg, MsgPath)
			hist := strToIntTab(msgHist)

			if slices.Contains(hist, MyId) {
				// j'ai déjà traité ce message
				rcvmsg = ""
				continue
			}

			switch rcvCat {
			case controleur:

				rcvData := findval(rcvmsg, MsgData)
				//Envoi de la donnée reçue au controleur avec le préfixe "NET:"
				fmt.Printf("NET:%s\n", rcvData)

				//Relai du message dans le réseau en mettant à jour l'historique
				fmt.Println(majHistorique(rcvmsg))
				break

			case admResponse:
				// Je ne traite pas les réponses d'admission car je suis déjà démarré
				break
			}
			rcvmsg = ""

		} else {
			// Le message vient de mon controleur

			if len(rcvmsg) > 11 && rcvmsg[:11] == "Blockchain:" {
				blockchainData = rcvmsg[11:]
				rcvmsg = ""
				continue
			} else if len(rcvmsg) > 6 && rcvmsg[:6] == "Queue:" {
				queueData = rcvmsg[6:]
				rcvmsg = ""
				continue
			} else if len(rcvmsg) > 12 && rcvmsg[:12] == "Controleurs:" {
				controllerNames = rcvmsg[12:]
				rcvmsg = ""
				continue
			}

			a := make([]int, 2)
			a[0] = MyId
			a[1] = MyId
			newMessage := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
				MsgFormat(MsgCategory, controleur) +
				MsgFormat(MsgPath, intTabToStr(a)) +
				MsgFormat(MsgData, rcvmsg)
			fmt.Println(newMessage)
			rcvmsg = ""
		}
	}
}
