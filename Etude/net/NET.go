package main

/*
Ce fichier contient les variables et les fonctions propres au fonctionnement d'un NET dans le cadre de notre application.
Le NET joue un rôle de médiateur entre le contrôleur et les autres sites du réseau réparti.
Il permet de mettre en place une arborescence bidirectionnelle dans le réseau, d'éviter les boucles et
que les mêmes messages soient traités plusieurs fois. Il permet également l'ajout d'un nouveau site au réseau, ainsi qu'un départ.
Pour cela, il implémente l'algorithme d'élection par extinction de vague.
Le NET est le premier à s'initialiser, il transmet ensuite le nombre de sites dans le réseau au contrôleur.
*/

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

// Caractères utilisés pour formater les messages entre les sites
// ils doivent être différents de ceux du contrôleur
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

// Constantes utilisées pour définir les champs de messages et certaines valeurs
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
	pid             = os.Getpid()
	stderr          = log.New(os.Stderr, "", 0)
	Nom             string  // Nom du NET : flag + PID
	MyId            = pid   // ID du NET : PID
	NbSites         = -1    // Nombre de sites total dans le réseau
	NbVoisins       int     // Nombre de voisins
	ListVoisins     []int   // Liste contenant les ID des voisins
	waitingSite     int     // ID du site en ayant réalisé une demande d'admission
	elecDisabled    = false // Indique si les elections sont temporairement désactivé pour ce site
	blockchainData  = ""    // Copie à jour de la blockchain sérialisée
	queueData       = ""    // Copie à jour de la file d'attente du contrôleur
	controllerNames = ""    // Copie à jour de la liste des noms des contrôleurs du réseau
)

var mutex = &sync.Mutex{}

// MsgFormat construit une partie de message formatée avec une clé et une valeur
func MsgFormat(key string, val string) string {
	return fieldsep + keyvalsep + key + keyvalsep + val
}

// findval extrait la valeur correspondant à une clé dans un message formaté
// Renvoi une chaine vide si la clé n'est pas trouvée
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

// initialisation permet de créer une arborescence dans le réseau et de compter le nombre total de sites
// Elle est appelée au démarrage du NET, lorsque aucun réseau n'existe déjà
func initialisation() {
	// Canal permettant à la goroutine receiveInit d'indiquer qu'elle a terminé
	var endInit = make(chan bool)

	stderr.Println(Nom, "Initialisation")

	// Envoi de son ID à ses voisins
	fmt.Println(MyId)
	// Lancement de la goroutine receiveInit, recevant et traitant les messages de l'initialisation
	go receiveInit(endInit)

	// Délai d'attente de découverte des voisins
	time.Sleep(time.Duration(2) * time.Second)

	mutex.Lock()
	// Une fois le délai passé, je compte mon nombre de voisins
	NbVoisins = len(ListVoisins)
	mutex.Unlock()

	//stderr.Println(Nom, "Fin des voisins", ListVoisins)

	mutex.Lock()
	// Démarrage d'une élection par extinction de vague pour définir l'arborescence
	// et compter le nombre de sites
	DemarrerElectionInit()
	mutex.Unlock()

	// Attente que la goroutine receiveInit soit terminée (arborescence finie)
	<-endInit
	stderr.Println(Nom, "Fin de l'initialisation")

	// Envoi du signal de départ au contrôleur en lui indiquant le nombre de sites
	fmt.Printf("NET:start:%d\n", NbSites)

	// On réinitialise les variables (hors parent/enfants) pour la prochaine election (ajout/départ)
	resetElection()
}

// initialisation reçoit et traite les messages liés à l'initialisation.
// Elle enregistre les ID envoyés par les voisins, puis traite les messages de l'élection.
func receiveInit(end chan bool) {
	var received string
	for {
		fmt.Scanln(&received)

		mutex.Lock()
		// Récupération de la catégorie du message
		msgCat := findval(received, MsgCategory)
		switch msgCat {
		// Le message est de catégorie élection
		case electionInit:

			for NbVoisins == 0 {
				// Je ne traite pas le message d'élection tant que je n'ai pas fini de compter mes voisins
				mutex.Unlock()
				time.Sleep(time.Duration(100) * time.Millisecond)
				mutex.Lock()
			}
			recevoirMessageElectionInit(received)
			break

		default:
			// Le message reçu n'a pas de catégorie, il s'agit d'un ID d'un voisin
			idVoisin, err := strconv.Atoi(received)
			if err != nil {
				continue
			}
			// Ajout de l'ID du voisin à la liste
			ListVoisins = append(ListVoisins, idVoisin)
			break
		}
		// L'élection n'est pas terminée tant que je n'ai pas récupéré le nombre de sites
		if NbSites != -1 {
			break
		}
		received = ""
		mutex.Unlock()
	}
	if win {
		// J'ai remporté l'élection, je transmets le nombre de sites à mes enfants
		envoyerA(electionInit, nombreSites, strconv.Itoa(NbSites), enfants)
	}
	// Indique la fin de l'initialisation
	end <- true
}

// demanderAdmission permet à un NET de rejoindre un réseau de NET déjà établi.
// Elle envoie une demande d'admission toutes les 5 secondes jusqu'à recevoir une confirmation.
func demanderAdmission() {
	stderr.Println(vert, "\n["+Nom+"]", "Demande d'admission", raz)

	// Envoi de la demande d'admission initiale
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
		MsgFormat(MsgCategory, admission)
	fmt.Println(msg)

	var received string
	heureDebut := time.Now()
	for {
		fmt.Scanln(&received)

		// Si le message reçu est une confirmation d'admission
		if findval(received, MsgCategory) == admResponse {

			// J'ajoute mon voisin comme parent
			parent, _ = strconv.Atoi(findval(received, MsgSender))
			NbVoisins = 1

			// Récupère les données de la part du parent
			blockchainData = findval(received, "blockchain")
			NbSites, _ = strconv.Atoi(findval(received, nombreSites))
			queueData = findval(received, "queue")
			controllerNames = findval(received, "controllerNames")

			// Indique au controller de lancer son initialisation
			fmt.Printf("NET:start:%d\n", NbSites)

			stderr.Println(vert, "["+Nom+"]", "Données reçues", raz)

			// Envoie les données nécessaires à l'initialisation du contrôleur
			fmt.Printf("NET:controleur:%s\n", controllerNames)
			fmt.Printf("NET:queue:%s\n", queueData)
			fmt.Printf("NET:blockchain:%s\n", blockchainData)

			// Election désactivée tant que le contrôleur n'a pas fini son initialisation
			elecDisabled = true

			break
		}
		// Si ne j'ai pas reçu de confirmation, je renvoie le message d'admission toutes les 5 secondes
		if time.Since(heureDebut) > 5*time.Second {
			fmt.Println(msg)
			heureDebut = time.Now()
		}
	}
}

// finaliserAdmission est appelée à la suite d'une demande d'admission par un site tier.
// Une fois l'élection remportée, cette fonction envoie la confirmation d'admission nouveau site,
// ainsi que les données nécessaires à son fonctionnement (NbSite, blockchain, fileAttente, listes des contrôleurs).
func finaliserAdmission(senderID int) {
	stderr.Println(magenta, "["+Nom+"]", "Finalisation d'admission", raz)

	//Ajout du demandeur à la liste des enfants et des voisins
	enfants = append(enfants, senderID)
	ListVoisins = append(ListVoisins, senderID)
	NbSites++
	NbVoisins++

	//Envoi au nouveau site les donées nécessaires à son initialisation
	infos := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
		MsgFormat(MsgCategory, admResponse) +
		MsgFormat("blockchain", blockchainData) +
		MsgFormat("queue", queueData) +
		MsgFormat("controllerNames", controllerNames) +
		MsgFormat(nombreSites, strconv.Itoa(NbSites))
	fmt.Println(infos)

	// Réinitialisation des variables pour la prochaine élection
	resetElection()

	// Envoi d'une confirmation d'admission au reste du réseau, permettant de mettre à jour le nombre de sites
	msg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, admConfirm) +
		MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
		MsgFormat(MsgData, strconv.Itoa(NbSites))

	fmt.Println(msg)
}

// majHistorique permet de mettre à jour l'historique d'un message reçu avant de le renvoyer.
// L'historique indique les deux derniers sites ayant traité le message, il permet d'éviter à ce que
// les messages soient traités en double.
func majHistorique(msg string) string {
	// Récupération de l'historique
	hist := strToIntTab(findval(msg, MsgPath))

	if len(hist) < 2 {
		stderr.Println(Nom, "Erreur, historique vide", hist)
		stderr.Println(Nom, msg)
	}
	// Décale les valeurs du tableau vers la gauche, et ajoute mon ID à la fin
	hist[0] = hist[1]
	hist[1] = MyId
	// Formattage du nouveau message avec l'historique mis à jour
	newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, findval(msg, MsgCategory)) +
		MsgFormat(MsgPath, intTabToStr(hist)) + MsgFormat(MsgData, findval(msg, MsgData))
	return newMsg
}

// Fonction principale du NET, elle lit les messages entrants et les traite dans une boucle infinie
func main() {

	flag.Parse()

	// Timer utilisé en cas de départ du site
	// pour relancer une élection toutes les 5 secondes
	var quitTimer time.Time

	Nom = *pNom + "-" + strconv.Itoa(pid)

	// Vérification s'il s'agit d'un ajout de site ou d'une intialisation classique
	if *pNouveauSite {
		time.Sleep(time.Duration(500) * time.Millisecond)
		demanderAdmission()
	} else {
		initialisation()
	}

	var rcvmsg string
	var fin bool = false
	var quit bool = false

	for !fin {

		fmt.Scanln(&rcvmsg)

		// Les messages de moins de 5 caractères sont rejettés
		if len(rcvmsg) < 5 {
			// Sauf s'il s'agit du signal de fin du controleur
			if rcvmsg == "fin" {

				quit = true
				// Démarrage d'une élection
				DemarrerElection()
				quitTimer = time.Now()
				continue
			}
			stderr.Println(Nom, "message trop court : "+rcvmsg)
			continue
		}
		// Le processus de départ est démarré depuis plus de 5 secondes
		if quit && time.Since(quitTimer) > 5*time.Second {
			// Relance une élection
			DemarrerElection()
			quitTimer = time.Now()
		}

		// Ignore les messages à destinations des contrôleurs ou de l'application
		if rcvmsg[:5] == "CONT:" || rcvmsg[:4] == "NET:" {
			//Ce message n'était pas à destination du NET
			rcvmsg = ""
			continue
		}

		// Traitement des messages
		rcvCat := findval(rcvmsg, MsgCategory)

		if rcvCat != "" {
			// Le message vient d'un autre NET et dispose d'une catégorie

			// Récupération de l'expéditeur
			msgSdr := findval(rcvmsg, MsgSender)
			sdrId, _ := strconv.Atoi(msgSdr)

			// Si le message ne vient pas de mon parent ou de mes enfants, je le rejette
			// Exception pour les demandes d'admission et les élections
			if (sdrId != parent && !slices.Contains(enfants, sdrId)) || rcvCat == election || rcvCat == admission {

				switch rcvCat {

				case admission:
					stderr.Println(orange, "["+Nom+"]", "admission reçu, tentative election", raz)
					// J'ai reçu une demande d'admission, je démarre une élection
					DemarrerElection()
					waitingSite = sdrId
					break

				case election:
					// Je traite le message d'élection si je ne l'ai pas désactivé
					if !elecDisabled {
						recevoirMessageElection(rcvmsg)
					}
					// J'ai gagné l'élection et je suis en processus de départ
					if win && quit {
						fin = true
						var args []string

						// Si je ne suis pas la racine de l'arbre
						if parent != MyId {
							// J'envoie un message indiquant mon départ ainsi que mon parent et mes enfants
							newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, outConfirm) +
								MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
								MsgFormat("children", intTabToStr(enfants)) + MsgFormat("parent", strconv.Itoa(parent)) +
								MsgFormat(MsgData, strconv.Itoa(MyId))
							fmt.Println(newMsg)

							// Argument du script de départ (mise à jour des fifo)
							args = []string{strconv.Itoa(parent)}
							for _, v := range enfants {
								args = append(args, strconv.Itoa(v))
							}

						} else {
							// Je suis la racine de l'arbre, j'indique mon premier enfant comme nouvelle racine
							newMsg := MsgFormat(MsgSender, strconv.Itoa(MyId)) + MsgFormat(MsgCategory, outConfirm) +
								MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
								MsgFormat("children", intTabToStr(enfants[1:])) + MsgFormat("parent", strconv.Itoa(enfants[0])) +
								MsgFormat(MsgData, strconv.Itoa(MyId))
							fmt.Println(newMsg)

							// Argument du script de départ (mise à jour des fifo)
							args = []string{strconv.Itoa(enfants[0])}
							for _, v := range enfants[1:] {
								args = append(args, strconv.Itoa(v))
							}
						}
						// Execution du script pour les fifo
						cmd := exec.Command("./quit.sh", args...)

						cmd.Stdout = os.Stderr
						cmd.Stderr = os.Stderr

						// Exécute le script Bash
						cmd.Run()

					} else if win {
						// J'ai remporté l'élection d'ajout d'un nouveau site
						finaliserAdmission(waitingSite)
					}
					break
				}

				rcvmsg = ""
				continue
			}

			// Le message provient de mon parent ou de mes enfants
			// Je récupère l'historique
			msgHist := findval(rcvmsg, MsgPath)
			hist := strToIntTab(msgHist)

			// Si mon ID est dans l'historique, je ne traite pas le message
			if slices.Contains(hist, MyId) {
				// j'ai déjà traité ce message
				rcvmsg = ""
				continue
			}

			switch rcvCat {

			// Le message concerne mon contrôleur
			case controleur:
				rcvData := findval(rcvmsg, MsgData)

				//Envoi de la donnée reçue au contrôleur avec le préfixe "NET:"
				fmt.Printf("NET:%s\n", rcvData)

				//Relai du message dans le réseau en mettant à jour l'historique
				fmt.Println(majHistorique(rcvmsg))
				break

			// Le message est une confirmation d'admission
			case admConfirm:
				// Mise à jour du nombre de sites
				NbSites, _ = strconv.Atoi(findval(rcvmsg, MsgData))
				// Propagation du message
				fmt.Println(majHistorique(rcvmsg))

				stderr.Println(blanc, "["+Nom+"]", "Fin de l'admission, nouveau nombre de sites :", NbSites, raz)

				// Réinitialisation de l'élection
				resetElection()
				break

			// Le message est une confirmation de départ
			case outConfirm:
				// Mise à jour du nombre de sites
				NbSites--
				stderr.Println(blanc, "["+Nom+"]", "Départ confirmé, nouveau nombre de sites :", NbSites, raz)

				// Récupération de l'ID du site partant
				idQuit, _ := strconv.Atoi(findval(rcvmsg, MsgData))

				// Si le site partant est mon parent
				if idQuit == parent {

					// Récupération du nouveau parent
					parent, _ = strconv.Atoi(findval(rcvmsg, "parent"))

					// Si le nouveau parent est différent de mon ID
					if parent != MyId {
						// Je ne suis pas la nouvelle racine de l'arbre

						// J'ajoute mon nouveau parent à la liste de mes voisins
						ListVoisins = append(ListVoisins, parent)

						// Je supprime mon ancien parent de la liste de mes voisins
						quitIndex := slices.Index(ListVoisins, idQuit)
						ListVoisins = append(ListVoisins[:quitIndex], ListVoisins[quitIndex+1:]...)

						stderr.Println(magenta, "["+Nom+"]", "Nouveau Parent", parent, raz)

					} else {
						// Je suis la nouvelle racine de l'arbre

						// Je récupère mes nouveaux enfants
						sdrChild := strToIntTab(findval(rcvmsg, "children"))

						// Si les enfants existent, je les ajoute à ma liste
						if len(sdrChild) > 0 {
							enfants = append(enfants, sdrChild...)
							ListVoisins = append(ListVoisins, sdrChild...)
						}

						// Je supprime mon ancien parent de la liste de mes voisins
						quitIndex := slices.Index(ListVoisins, idQuit)
						ListVoisins = append(ListVoisins[:quitIndex], ListVoisins[quitIndex+1:]...)

						stderr.Println(magenta, "["+Nom+"]", "Nouveaux enfants", enfants, raz)

						// Je mets à jour mon nombre de voisins
						NbVoisins += len(sdrChild) - 1
					}

				} else if slices.Contains(enfants, idQuit) {
					// Le site partant est mon enfant

					// Je récupère la liste de ses enfants
					sdrChild := strToIntTab(findval(rcvmsg, "children"))

					// Si les enfants existent, je les ajoute à ma liste
					if len(sdrChild) > 0 {
						enfants = append(enfants, sdrChild...)
						ListVoisins = append(ListVoisins, sdrChild...)

						// Je supprime mon ancien enfant de la liste de mes enfants
						quitIndex := slices.Index(enfants, idQuit)
						enfants = append(enfants[:quitIndex], enfants[quitIndex+1:]...)
					}
					stderr.Println(magenta, "["+Nom+"]", "Nouveaux enfants", enfants, raz)

					// Je supprime mon ancien enfant de la liste de mes voisins
					quitIndex := slices.Index(ListVoisins, idQuit)
					ListVoisins = append(ListVoisins[:quitIndex], ListVoisins[quitIndex+1:]...)

					// Je mets à jour mon nombre de voisins
					NbVoisins += len(sdrChild) - 1

				} else if slices.Contains(ListVoisins, idQuit) {
					// Le site partant est simplement un voisin

					// Je mets à jour mon nombre de voisins
					NbVoisins--

					// Je le supprime de la liste de mes voisins
					quitIndex := slices.Index(ListVoisins, idQuit)
					ListVoisins = append(ListVoisins[:quitIndex], ListVoisins[quitIndex+1:]...)
				}

				// Je propage le message dans le réseau, en mettant à jour l'historique
				fmt.Println(majHistorique(rcvmsg) + MsgFormat("parent", findval(rcvmsg, "parent")) +
					MsgFormat("children", findval(rcvmsg, "children")))

				// Sleep pour laisser le temps aux fifos de changer
				time.Sleep(time.Duration(5) * time.Second)

				// Réinitialisation de l'élection
				resetElection()
				break
			}

			rcvmsg = ""

		} else {
			// Le message vient de mon contrôleur

			// Le message est une copie de données à mettre à jour
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
				// Le message indique que le contrôleur a fini son initialisation
				time.Sleep(time.Duration(100) * time.Millisecond)
				// J'accepte les prochaines élections
				elecDisabled = false
				continue
			}

			// Je formate le message et l'envoi dans le réseau
			newMessage := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
				MsgFormat(MsgCategory, controleur) +
				MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
				MsgFormat(MsgData, rcvmsg)
			fmt.Println(newMessage)

			rcvmsg = ""
		}
	}
	// Le NET s'arrête
	stderr.Println(rougec, "["+Nom+"]", "Fin du NET\n", raz)
	for {
		// Boucle pour continuer à lire les messages (empêcher le blocage de la fifo)
		fmt.Scanln(&rcvmsg)
		rcvmsg = ""
	}
}
