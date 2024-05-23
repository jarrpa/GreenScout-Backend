package constants

var CachedConfigs GeneralConfigs

var CustomEventKey bool = false

type GeneralConfigs struct {
	PythonDriver       string       `yaml:"PythonDriver"`
	SqliteDriver       string       `yaml:"SqliteDriver"`
	TBAKey             string       `yaml:"TBAKey"`
	EventKey           string       `yaml:"EventKey"`
	EventKeyName       string       `yaml:"EventKeyName"`
	IP                 string       `yaml:"IP"`
	DomainName         string       `yaml:"DomainName"`
	UsingMultiScouting bool         `yaml:"UsingMultiScouting"`
	SpreadSheetID      string       `yaml:"SpreadSheetID"`
	PathToDatabases    string       `yaml:"PathToDatabases"`
	SlackConfigs       SlackConfigs `yaml:"SlackConfigs"`
}

type SlackConfigs struct {
	Configured bool   `yaml:"Configured"`
	UsingSlack bool   `yaml:"UsingSlack"`
	BotToken   string `yaml:"Token"`
	Channel    string `yaml:"Channel"`
}
