package constants

var CachedConfigs GeneralConfigs

type GeneralConfigs struct {
	PythonDriver       string `yaml:"PythonDriver"`
	SqliteDriver       string `yaml:"SqliteDriver"`
	TBAKey             string `yaml:"TBAKey"`
	EventKey           string `yaml:"EventKey"`
	EventKeyName       string `yaml:"EventKeyName"`
	IP                 string `yaml:"IP"`
	DomainName         string `yaml:"DomainName"`
	UsingMultiScouting bool   `yaml:"UsingMultiScouting"`
	SpreadSheetID      string `yaml:"SpreadSheetID"`
	PathToDatabases    string `yaml:"PathToDatabases"`
}
