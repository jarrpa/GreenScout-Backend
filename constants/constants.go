package constants

import "path/filepath"

var CachedConfigs GeneralConfigs

var CustomEventKey bool = false

var DefaultPfpPath = filepath.Join("pfp", "pictures", "Default_pfp.png")

type GeneralConfigs struct {
	PythonDriver       string             `yaml:"PythonDriver"`
	SqliteDriver       string             `yaml:"SqliteDriver"`
	TBAKey             string             `yaml:"TBAKey"`
	EventKey           string             `yaml:"EventKey"`
	EventKeyName       string             `yaml:"EventKeyName"`
	CustomEventConfigs CustomEventConfigs `yaml:"CustomEventConfigs"`
	IP                 string             `yaml:"IP"`
	DomainName         string             `yaml:"DomainName"`
	UsingMultiScouting bool               `yaml:"UsingMultiScouting"`
	SpreadSheetID      string             `yaml:"SpreadSheetID"`
	PathToDatabases    string             `yaml:"PathToDatabases"`
	SlackConfigs       SlackConfigs       `yaml:"SlackConfigs"`
}

type SlackConfigs struct {
	Configured bool   `yaml:"Configured"`
	UsingSlack bool   `yaml:"UsingSlack"`
	BotToken   string `yaml:"Token"`
	Channel    string `yaml:"Channel"`
}

type CustomEventConfigs struct {
	Configured     bool `yaml:"Configured"`
	CustomSchedule bool `yaml:"CustomSchedule"`
}
