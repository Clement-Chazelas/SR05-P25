package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var fieldsep = "?"
var keyvalsep = "@"

// Listes des couleurs pour l'affichage dans la console
var noir string = "\033[1;30m"
var rougec string = "\033[1;31m"
var vert string = "\033[1;32m"
var orange string = "\033[1;33m"
var magenta string = "\033[1;35m"
var cyan string = "\033[1;36m"
var blanc string = "\033[1;37m"
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
	admConfirm     = "admcf"
	outConfirm     = "outcf"
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
	elecDisabled    bool = false
	end             bool = false
)

var mutex = &sync.Mutex{}

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
	stderr.Println(vert, "\n["+Nom+"]", "Demande d'admission", raz)
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
		MsgFormat(MsgCategory, admission)
	fmt.Println(msg)

	var received string
	heureDebut := time.Now()
	for {
		fmt.Scanln(&received)
		if findval(received, MsgCategory) == admResponse {
			// J'ajoute mon voisin comme parent
			parent, _ = strconv.Atoi(findval(received, MsgSender))
			NbVoisins = 1

			//On récupère les données de la part du parent
			blockchainData = findval(received, "blockchain")
			NbSites, _ = strconv.Atoi(findval(received, nombreSites))
			queueData = findval(received, "queue")
			controllerNames = findval(received, "controllerNames")

			//On dit au controller de lancer son initialisation
			fmt.Printf("NET:start:%d\n", NbSites)
			stderr.Println(vert, "["+Nom+"]", "Données reçues", raz)
			//On envoie les données nécessaires à l'initialisation du contrôleur
			fmt.Printf("NET:controleur:%s\n", controllerNames)
			fmt.Printf("NET:queue:%s\n", queueData)
			fmt.Printf("NET:blockchain:%s\n", blockchainData)

			// Election désactivée tant que le controleur n'a pas fini son init
			elecDisabled = true

			break
		}
		if time.Since(heureDebut) > 5*time.Second {
			fmt.Println(msg)
			heureDebut = time.Now()
		}
	}
}

func finaliserAdmission(senderID int) {
	stderr.Println(magenta, "["+Nom+"]", "Finalisation d'admission", raz)
	//On ajoute le demandeur au parent en tant qu'enfant
	enfants = append(enfants, senderID)
	ListVoisins = append(ListVoisins, senderID)
	NbSites++
	NbVoisins++
	//On envoie au nouveau site les infos nécessaires à son initialisation
	infos := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
		MsgFormat(MsgCategory, admResponse) +
		MsgFormat("blockchain", blockchainData) +
		MsgFormat("queue", queueData) +
		MsgFormat("controllerNames", controllerNames) +
		MsgFormat(nombreSites, strconv.Itoa(NbSites))
	fmt.Println(infos)

	// Réinitialisation des variables pour la prochaine élection
	resetElection()

	// Envoi du nv nb de site
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, admConfirm) +
		MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
		MsgFormat(MsgData, strconv.Itoa(NbSites))

	fmt.Println(msg)
}

func initialisation() {
	var endInit = make(chan bool)
	stderr.Println(Nom, "Initialisation")

	fmt.Println(MyId)
	go receiveInit(endInit)

	// Délai d'attente de découverte des voisins
	time.Sleep(time.Duration(2) * time.Second)

	mutex.Lock()
	NbVoisins = len(ListVoisins)
	mutex.Unlock()

	//stderr.Println(Nom, "Fin des voisins", ListVoisins)

	mutex.Lock()
	DemarrerElectionInit()
	mutex.Unlock()

	<-endInit
	stderr.Println(Nom, "Fin de l'initialisation")
	fmt.Printf("NET:start:%d\n", NbSites)

	// On réinitialise les variables (hors parent/enfants) pour la prochaine election (ajout/départ)
	resetElection()
}

func receiveInit(end chan bool) {
	var received string
	for {
		fmt.Scanln(&received)
		mutex.Lock()
		msgCat := findval(received, MsgCategory)
		switch msgCat {
		case electionInit:
			// Je ne traite pas le message d'election tant que je n'ai pas fini de compter mes voisins
			for NbVoisins == 0 {
				mutex.Unlock()
				time.Sleep(time.Duration(100) * time.Millisecond)
				mutex.Lock()
			}
			recevoirMessageElectionInit(received)
			break

		default:
			idVoisin, err := strconv.Atoi(received)
			if err != nil {
				continue
			}
			ListVoisins = append(ListVoisins, idVoisin)
			break
		}
		mutex.Unlock()
		if NbSites != -1 {
			break
		}
		received = ""
	}
	if win {
		// J'ai remporté l'élection, je transmets le nb de site à mes enfants
		envoyerA(electionInit, nombreSites, strconv.Itoa(NbSites), enfants)
	}
	end <- true
}

func majHistorique(msg string) string {
	hist := strToIntTab(findval(msg, MsgPath))
	if len(hist) == 1 {
		stderr.Println(Nom, "Erreur, historique vide", hist)
		stderr.Println(Nom, msg)
	}
	hist[0] = hist[1]
	hist[1] = MyId
	newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, findval(msg, MsgCategory)) +
		MsgFormat(MsgPath, intTabToStr(hist)) + MsgFormat(MsgData, findval(msg, MsgData))
	return newMsg
}

func main() {
	flag.Parse()
	Nom = *pNom + "-" + strconv.Itoa(pid)

	//On check si c'est un nouveau site ou pas, si oui on demande l'admission, si non alors on s'initialise normalement
	if *pNouveauSite {
		time.Sleep(time.Duration(500) * time.Millisecond)
		demanderAdmission()
	} else {
		initialisation()
	}

	var rcvmsg string
	var fin bool = false
	for !fin {
		fmt.Scanln(&rcvmsg)

		if len(rcvmsg) < 5 {
			if rcvmsg == "fin" {
				DemarrerElection()
				end = true
				continue
			}
			stderr.Println(Nom, "message trop court : "+rcvmsg)
			continue
		}

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
					stderr.Println(orange, "["+Nom+"]", "admission reçu, tentative election", raz)
					DemarrerElection()
					waitingSite = sdrId
					break
				case election:
					if !elecDisabled {
						recevoirMessageElection(rcvmsg)
					}
					if win && end {
						fin = true
						var args []string
						// Si je ne suis pas la racine de l'arbre
						if parent != MyId {
							newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, outConfirm) +
								MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
								MsgFormat("children", intTabToStr(enfants)) + MsgFormat("parent", strconv.Itoa(parent)) +
								MsgFormat(MsgData, strconv.Itoa(MyId))
							fmt.Println(newMsg)

							args = []string{strconv.Itoa(parent)}
							for _, v := range enfants {
								args = append(args, strconv.Itoa(v))
							}

						} else {
							// Je suis la racine de l'arbre, la nouvelle racine est mon premier enfant
							newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, outConfirm) +
								MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
								MsgFormat("children", intTabToStr(enfants)) + MsgFormat("parent", strconv.Itoa(enfants[0])) +
								MsgFormat(MsgData, strconv.Itoa(MyId))
							fmt.Println(newMsg)

							args = []string{strconv.Itoa(enfants[0])}
							for _, v := range enfants[1:] {
								args = append(args, strconv.Itoa(v))
							}
						}
						cmd := exec.Command("./quit.sh", args...)

						cmd.Stdout = os.Stderr
						cmd.Stderr = os.Stderr

						// Exécute le script Bash
						cmd.Run()

					} else if win {
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

			case election:
				if !elecDisabled {
					recevoirMessageElection(rcvmsg)
				}
				if win && end {
					fin = true
					var args []string
					// Si je ne suis pas la racine de l'arbre
					if parent != MyId {
						newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, outConfirm) +
							MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
							MsgFormat("children", intTabToStr(enfants)) + MsgFormat("parent", strconv.Itoa(parent)) +
							MsgFormat(MsgData, strconv.Itoa(MyId))
						fmt.Println(newMsg)

						args = []string{strconv.Itoa(parent)}
						for _, v := range enfants {
							args = append(args, strconv.Itoa(v))
						}

					} else {
						// Je suis la racine de l'arbre, la nouvelle racine est mon premier enfant
						newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, outConfirm) +
							MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
							MsgFormat("children", intTabToStr(enfants[1:])) + MsgFormat("parent", strconv.Itoa(enfants[0])) +
							MsgFormat(MsgData, strconv.Itoa(MyId))
						fmt.Println(newMsg)

						args = []string{strconv.Itoa(enfants[0])}
						for _, v := range enfants[1:] {
							args = append(args, strconv.Itoa(v))
						}
					}

					time.Sleep(time.Duration(1) * time.Second)
					cmd := exec.Command("./quit.sh", args...)

					cmd.Stdout = os.Stderr
					cmd.Stderr = os.Stderr

					// Exécute le script Bash
					cmd.Run()

				} else if win {
					finaliserAdmission(waitingSite)
				}
				break

			case admConfirm:
				NbSites, _ = strconv.Atoi(findval(rcvmsg, MsgData))
				fmt.Println(majHistorique(rcvmsg))
				stderr.Println(blanc, "["+Nom+"]", "Fin de l'admission, nouveau nombre de sites :", NbSites, raz)
				resetElection()
				break

			case outConfirm:
				NbSites--
				stderr.Println(blanc, "["+Nom+"]", "Départ confirmé, nouveau nombre de sites :", NbSites, raz)
				idQuit, _ := strconv.Atoi(findval(rcvmsg, MsgData))
				if idQuit == parent {
					parent, _ = strconv.Atoi(findval(rcvmsg, "parent"))

					if parent != MyId {
						ListVoisins = append(ListVoisins, parent)

						quitIndex := slices.Index(ListVoisins, idQuit)
						ListVoisins = append(ListVoisins[:quitIndex], ListVoisins[quitIndex+1:]...)

						stderr.Println(magenta, "["+Nom+"]", "Nouveau Parent", parent, raz)

					} else {

						sdrChild := strToIntTab(findval(rcvmsg, "children"))
						// Je suis la nouvelle racine de l'arbre
						if len(sdrChild) > 0 {
							enfants = append(enfants, sdrChild...)
							ListVoisins = append(ListVoisins, sdrChild...)

						}

						quitIndex := slices.Index(ListVoisins, idQuit)
						ListVoisins = append(ListVoisins[:quitIndex], ListVoisins[quitIndex+1:]...)

						stderr.Println(magenta, "["+Nom+"]", "Nouveaux enfants", enfants, raz)

						NbVoisins += len(sdrChild) - 1
					}
				} else if slices.Contains(enfants, idQuit) {
					sdrChild := strToIntTab(findval(rcvmsg, "children"))
					if len(sdrChild) > 0 {
						enfants = append(enfants, sdrChild...)
						ListVoisins = append(ListVoisins, sdrChild...)

						quitIndex := slices.Index(enfants, idQuit)
						enfants = append(enfants[:quitIndex], enfants[quitIndex+1:]...)
					}
					stderr.Println(magenta, "["+Nom+"]", "Nouveaux enfants", enfants, raz)

					quitIndex := slices.Index(ListVoisins, idQuit)
					ListVoisins = append(ListVoisins[:quitIndex], ListVoisins[quitIndex+1:]...)

					NbVoisins += len(sdrChild) - 1
				} else if slices.Contains(ListVoisins, idQuit) {
					NbVoisins--
					quitIndex := slices.Index(ListVoisins, idQuit)
					ListVoisins = append(ListVoisins[:quitIndex], ListVoisins[quitIndex+1:]...)
				}
				fmt.Println(majHistorique(rcvmsg) + MsgFormat("parent", findval(rcvmsg, "parent")) +
					MsgFormat("children", findval(rcvmsg, "children")))

				time.Sleep(time.Duration(5) * time.Second)
				resetElection()
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
			} else if rcvmsg == "FinInit" {
				time.Sleep(time.Duration(100) * time.Millisecond)
				elecDisabled = false
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
	stderr.Println(rougec, "["+Nom+"]", "Fin du NET\n", raz)
	for {
		fmt.Scanln(&rcvmsg)
		rcvmsg = ""
	}
}
