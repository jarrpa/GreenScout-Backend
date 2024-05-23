package main

import (
	"GreenScoutBackend/constants"
	filemanager "GreenScoutBackend/fileManager"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/lib"
	"GreenScoutBackend/schedule"
	"GreenScoutBackend/server"
	"GreenScoutBackend/setup"
	"GreenScoutBackend/sheet"
	"GreenScoutBackend/userDB"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	greenlogger.InitLogFile()

	isSetup := slices.Contains(os.Args, "setup")

	if isSetup && filemanager.IsSudo() {
		greenlogger.FatalLogMessage("If you are running in setup mode, please run without sudo!")
	}

	setup.TotalSetup(slices.Contains(os.Args, "test")) //Allows setup to bypass ip and domain validation to run localhost

	sheet.WriteConditionalFormatting()
	if isSetup {
		os.Exit(1)
	}

	schedule.InitScoutDB()
	userDB.InitAuthDB()
	userDB.InitUserDB()

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

	inProduction := slices.Contains(os.Args, "prod") && !slices.Contains(os.Args, "test")

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
			greenlogger.FatalError(http.ListenAndServe(":http", h), "http.ListenAndServe() failed")
		}()

	} else {
		jSrv.Addr = ":8443"

		crtPath = "server.crt"
		keyPath = "server.key"
	}

	go func() {
		err := jSrv.ListenAndServeTLS(crtPath, keyPath)
		if err != nil {
			greenlogger.FatalError(err, "jSrv.ListendAndServeTLS() failed")
		}
	}()

	if inProduction {
		setup.EnsureExternalConnectivity()
	}

	greenlogger.LogMessage("Server Successfully Set Up!")
	if constants.CachedConfigs.SlackConfigs.UsingSlack {
		greenlogger.NotifyOnline(true)
	}

	go server.RunServerLoop()

	// Listen for termination signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-signalCh
	if constants.CachedConfigs.SlackConfigs.UsingSlack {
		greenlogger.NotifyOnline(false)
	}
}
