package main

import (
	"GreenScoutBackend/lib"
	"GreenScoutBackend/server"
	"GreenScoutBackend/sheet"
	"os"
)

func main() {
	sheet.SetupSheetsAPI()
	lib.RegisterTeamColumns(lib.GetCachedEventKey())

	if len(os.Args) > 1 && os.Args[1] == "setup" {
		sheet.FillOutTeamSheet("AllTeams")
	}

	jSrv := server.SetupServer()

	go jSrv.ListenAndServeTLS("server.crt", "server.key")

	println("Server successfully set up!")

	server.RunServerLoop()
}
