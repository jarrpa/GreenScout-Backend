package main

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/lib"
	"GreenScoutBackend/schedule"
	"GreenScoutBackend/server"
	"GreenScoutBackend/setup"
	"GreenScoutBackend/sheet"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	greenlogger.InitLogFile()

	setup.TotalSetup()

	schedule.InitScoutDB()

	if slices.Contains(os.Args, "matches") {
		var usingRemainder bool = false

		matches := lib.GetNumMatches()

		blocks := matches / 50

		var remainder int
		if remainder = matches % 50; remainder > 0 { // Remainder
			usingRemainder = true
		}

		for i := 1; i <= blocks*50; i += 50 {
			sheet.FillMatches(i, i+49)
			time.Sleep(1 * time.Minute)
		}

		if usingRemainder {
			initial := blocks * 50
			if initial == 0 {
				initial++
			}
			sheet.FillMatches(initial, initial+remainder)
		}
	}

	inProduction := slices.Contains(os.Args, "prod")

	jSrv := server.SetupServer()

	crtPath := ""
	keyPath := ""
	var serverManager *autocert.Manager
	if inProduction {
		serverManager = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(constants.CachedConfigs.DomainName),
			Cache:      autocert.DirCache("./Certs"), // This may not be the... wisest choice. Anyone in the future, feel free to fix.
		}
		jSrv.Addr = ":443"
		jSrv.TLSConfig = &tls.Config{GetCertificate: serverManager.GetCertificate}

		go func() {
			h := serverManager.HTTPHandler(nil)
			log.Fatal(http.ListenAndServe(":http", h))
		}()

	} else {
		crtPath = "server.crt"
		keyPath = "server.key"
	}

	go func() {
		err := jSrv.ListenAndServeTLS(crtPath, keyPath)
		if err != nil {
			log.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
		}
	}()

	setup.EnsureExternalConnectivity()

	print("Server Successfully Set Up!")

	server.RunServerLoop()
}
