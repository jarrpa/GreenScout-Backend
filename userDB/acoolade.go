package userDB

import (
	greenlogger "GreenScoutBackend/greenLogger"
	"encoding/json"
	"slices"
)

const ( // Accolade name enum
	Rookie      Accolade = "Scouting Rookie"     // Scouted 1
	Novice      Accolade = "Scouting Novice"     // Scouted 10
	Scouter     Accolade = "Scouter"             // Scouted 50
	Pro         Accolade = "Scouting Pro"        // Scouted 100
	Enthusiast  Accolade = "Scouting Enthusiast" // Scouted 500
	Locked      Accolade = "Locked In"           // High 50
	Deja        Accolade = "Déjà vu"             // High 78
	Eyes        Accolade = "👀"                   // YES THIS IS DUMB I KNOW; High 300
	Strategizer Accolade = "Strategizer"
	Foreign     Accolade = "Foreign Fracas"
	Detective   Accolade = "Detective"
	Debugger    Accolade = "Debugger"
	mnmi224     Accolade = "Mvp_2024mnmi2"
	Dev         Accolade = "App Dev"
	FDev        Accolade = "Frontend_Dev"
	BDev        Accolade = "Backend_Dev"
	StratLead   Accolade = "Strategy Lead"
	Lead        Accolade = "Leadership"
	Captain     Accolade = "Captain"
	AsstCaptain Accolade = "Assistant_Captain"
	CSPLead     Accolade = "CSP_Lead"
	MechLead    Accolade = "Mech_Lead"
	Mentor      Accolade = "Mentor"
	AppMentor   Accolade = "App_Mentor"
	Admin       Accolade = "Admin"
	Super       Accolade = "Super Admin"
	Test        Accolade = "Test"
	Driveteam   Accolade = "Driveteam"
	HOF         Accolade = "HOF"
	Bug         Accolade = "Bug_Finder"
	Early       Accolade = "Early"
	Router      Accolade = "Router Dungeon Survivor"
)

var AllAccolades = []Accolade{
	Rookie,
	Novice,
	Scouter,
	Pro,
	Enthusiast,
	Locked,
	Deja,
	Eyes,
	Strategizer,
	Foreign,
	Detective,
	Debugger,
	mnmi224,
	Dev,
	FDev,
	BDev,
	StratLead,
	Lead,
	Captain,
	AsstCaptain,
	CSPLead,
	MechLead,
	Mentor,
	AppMentor,
	Admin,
	Super,
	Test,
	Driveteam,
	HOF,
	Bug,
	Early,
	Router,
}

var frontendAchievements = []Accolade{
	Strategizer,
	Foreign,
	Detective,
	Debugger,
}

type FrontendAdds struct {
	UUID         string     `json:"UUID"`
	Achievements []Accolade `json:"Achievements"`
}

func ConsumeFrontendAdditions(adds FrontendAdds, isAdmin bool) {
	for _, add := range adds.Achievements {
		if isAdmin && slices.Contains(AllAccolades, add) {
			AddAccolade(adds.UUID, add, false)
		} else if slices.Contains(frontendAchievements, add) {
			AddAccolade(adds.UUID, add, true)
		}
	}
}

func GetAccolades(uuid string) []AccoladeData {
	var Accolades []AccoladeData
	var AccoladesMarshalled string
	response := userDB.QueryRow("select accolades from users where uuid = ?", uuid)
	scanErr := response.Scan(&AccoladesMarshalled)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning results of sql query SELECT accolades FROM users WHERE uuid = ? with arg: "+uuid)
	}
	// i am aware of how awful converting []byte -> string -> []byte is but i've had problems storing byte arrays with sqlite. postgres doesn't have this problem but what high schooler is learning postgres
	unmarshalErr := json.Unmarshal([]byte(AccoladesMarshalled), &Accolades)
	if unmarshalErr != nil {
		greenlogger.LogErrorf(unmarshalErr, "Problem unmarshalling %v", AccoladesMarshalled)
	}

	return Accolades
}

func ExtractNames(accolades []AccoladeData) []Accolade {
	var names []Accolade
	for _, accolade := range accolades {
		names = append(names, accolade.Accolade)
	}
	return names
}

func AccoladesHas(accolades []AccoladeData, accoladeToCheck Accolade) bool {
	for _, accolade := range accolades {
		if accolade.Accolade == accoladeToCheck {
			return true
		}
	}
	return false
}
