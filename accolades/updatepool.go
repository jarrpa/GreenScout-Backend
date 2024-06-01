package accolades

import (
	filemanager "GreenScoutBackend/fileManager"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/userDB"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
)

type UpdatePool struct {
	Path    string                    `json:"-"`
	Updates map[string][]Notification `json:"Update"`
}

type Notification struct {
	Accolade    userDB.Accolade `json:"Accolade"`
	MutDesc     bool            `json:"MutDesc"`
	Description string          `json:"Description"`
	NotifId     string          `json:"NotifId"`
}

const ( // Accolade name enum
	Rookie      userDB.Accolade = "Scouting Rookie"
	Novice      userDB.Accolade = "Scouting Novice"
	Scouter     userDB.Accolade = "Scouter"
	Pro         userDB.Accolade = "Scouting Pro"
	Enthusiast  userDB.Accolade = "Scouting Enthusiast"
	Locked      userDB.Accolade = "Locked In"
	Deja        userDB.Accolade = "Deja Vu"
	Eyes        userDB.Accolade = "Eyes"
	Strategizer userDB.Accolade = "Strategizer"
	Foreign     userDB.Accolade = "Foreign Fracas"
	Detective   userDB.Accolade = "Detective"
	Debugger    userDB.Accolade = "Debugger"
	mnmi224     userDB.Accolade = "Mvp_2024mnmi2"
	Dev         userDB.Accolade = "Developer"
	FDev        userDB.Accolade = "Frontend_Dev"
	BDev        userDB.Accolade = "Backend_Dev"
	StratLead   userDB.Accolade = "Strategy Lead"
	Lead        userDB.Accolade = "Leadership"
	Captain     userDB.Accolade = "Captain"
	AsstCaptain userDB.Accolade = "Assistant_Captain"
	CSPLead     userDB.Accolade = "CSP_Lead"
	MechLead    userDB.Accolade = "Mech_Lead"
	Mentor      userDB.Accolade = "Mentor"
	AppMentor   userDB.Accolade = "App_Mentor"
	Admin       userDB.Accolade = "Admin"
	Super       userDB.Accolade = "SuperAdmin"
	Test        userDB.Accolade = "Test"
	Driveteam   userDB.Accolade = "Driveteam"
	HOF         userDB.Accolade = "HOF"
	Bug         userDB.Accolade = "Bug_Finder"
	Early       userDB.Accolade = "Early"
	Router      userDB.Accolade = "Router"
)

var frontendAchievements = []userDB.Accolade{
	Strategizer,
	Foreign,
	Detective,
	Debugger,
}

var pool UpdatePool
var poolPath = filepath.Join("accolades", "updatePool.json")

func BeginUpdatePool() {
	file, err := filemanager.OpenWithPermissions(poolPath)
	if err != nil {
		greenlogger.LogError(err, "Problem opening "+poolPath)
	}
	defer file.Close()

	decodeErr := json.NewDecoder(file).Decode(&pool)
	if decodeErr != nil && !strings.Contains(decodeErr.Error(), "EOF") {
		greenlogger.LogError(decodeErr, "Problem decoding "+poolPath)
	}

	pool.Path = poolPath

	encodeErr := json.NewEncoder(file).Encode(&pool)
	if encodeErr != nil {
		greenlogger.LogErrorf(encodeErr, "Problem encoding %v to %v", pool, pool.Path)
	}
}

func ShutdownUpdatePool() {
	file, err := filemanager.OpenWithPermissions(poolPath)
	if err != nil {
		greenlogger.LogError(err, "Problem opening "+poolPath)
	}
	defer file.Close()

	encodeErr := json.NewEncoder(file).Encode(&pool)
	if encodeErr != nil {
		greenlogger.LogErrorf(encodeErr, "Problem encoding %v to %v", pool, pool.Path)
	}
}

func GetUserNotifs(uuid string) []Notification {
	return pool.Updates[uuid]
}

type Confirmation struct {
	UUID string   `json:"UUID"`
	IDs  []string `json:"NotifIDs"`
}

type FrontendAdds struct {
	UUID         string            `json:"UUID"`
	Achievements []userDB.Accolade `json:"Achievements"`
}

func ConsumeUpdateConfirmations(confirmation Confirmation) {
	var newNotifs []Notification
	for _, notification := range pool.Updates[confirmation.UUID] {
		if !slices.Contains(confirmation.IDs, notification.NotifId) {
			newNotifs = append(newNotifs, notification)
		}
	}

	pool.Updates[confirmation.UUID] = newNotifs
}

func ConsumeFrontendAdditions(adds FrontendAdds) {
	for _, add := range adds.Achievements {
		if slices.Contains(frontendAchievements, add) {
			userDB.AddAccolade(adds.UUID, add)
		}
	}
}
