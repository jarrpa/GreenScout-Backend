package config

import (
	"GreenScoutBackend/lib"
	"GreenScoutBackend/sheet"
	"encoding/json"
	"os"
)

func SetEventKey(key string, tab string) {
	name := lib.RegisterTeamColumns(key)

	file, _ := os.Create("config/GreenScoutConfig.json")
	json.NewEncoder(file).Encode(&lib.EventConfig{EventKey: key, EventName: name})

	lib.CachedKey = &key

	sheet.FillOutTeamSheet(tab)
}
