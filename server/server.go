package server

import (
	"GreenScoutBackend/auth"
	"GreenScoutBackend/config"
	"GreenScoutBackend/lib"
	"GreenScoutBackend/sheet"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func RunServerLoop() {
	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	func() {
		for {
			select {
			case <-ticker.C:
				go iterativeServerCall()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func iterativeServerCall() {
	allJson, _ := os.ReadDir("InputtedJson/In")

	// Avoid nil files
	if len(allJson) > 0 {
		// Only deal with first file
		file := allJson[0]

		// Parse and write to spreadsheet
		team := lib.Parse(file.Name())
		sheet.WriteTeamDataToLine(team, lib.GetRow(team))

		// Move written file out of written
		oldStr := "InputtedJson/In/" + file.Name()
		oldLoc, _ := os.Open(oldStr)

		newLoc, _ := os.Create("InputtedJson/Written/" + file.Name())
		defer newLoc.Close()

		io.Copy(newLoc, oldLoc)

		oldLoc.Close()

		os.Remove(oldStr)

		println("Successfully Processed " + file.Name())
	}
}

func SetupServer() *http.Server {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/pub", servePublicKey)
	http.HandleFunc("/schedule", handleScheduleRequest)

	http.HandleFunc("/login", handleLoginRequest)

	http.HandleFunc("/dataEntry", postJson)

	http.HandleFunc("/keyChange", handleKeyChange)
	http.HandleFunc("/sheetChange", handleSheetChange)

	jsrv := &http.Server{
		Addr:         "127.0.0.1:8443", //Localhost! Change when actually in use
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	return jsrv
}

func handleRoot(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "howdy!")
}

func postJson(writer http.ResponseWriter, request *http.Request) {
	_, authenticated := auth.VerifyCertificate(request.Header.Get("Certificate")) //Don't care about specific role for post, everyone that is auth'd can.

	if authenticated {
		decoder := json.NewDecoder(request.Body)

		var team lib.TeamData
		err := decoder.Decode(&team)
		if err != nil {
			panic(err)
		}

		fileName := fmt.Sprintf("%s_%v_%s", lib.GetCurrentEvent(), team.Match.Number, lib.GetDSString(team.DriverStation.IsBlue, uint(team.DriverStation.Number)))
		//EVENT_MATCH_{COLOR}{DSNUM}
		file, err := os.Create("InputtedJson/In/" + fileName + ".json")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.Encode(team)

		fmt.Fprintf(writer, "Processed %v\n", fileName)
	}
}

func handleKeyChange(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := auth.VerifyCertificate(request.Header.Get("Certificate")) //Don't care about specific role for post, everyone that is auth'd can.
	if authenticated && role == "super" {
		requestBytes, _ := io.ReadAll(request.Body)

		newKey := string(requestBytes)

		config.SetEventKey(newKey, "AllTeams")

		fmt.Fprintf(writer, "Successfully changed event key to "+newKey+"\n")
	} else if !authenticated {
		fmt.Fprintf(writer, "Not successfully authenticated. Please ensure you have correct login details.\n")
	} else {
		fmt.Fprintf(writer, "Not a super user. womp womp\n")
	}
}

func handleScheduleRequest(writer http.ResponseWriter, request *http.Request) {
	file, _ := os.Open("schedule/schedule.json")

	var scheduleJson any
	json.NewDecoder(file).Decode(&scheduleJson)
	json.NewEncoder(writer).Encode(&scheduleJson)
}

func handleLoginRequest(writer http.ResponseWriter, request *http.Request) {
	var loginRequest auth.LoginAttempt

	json.NewDecoder(request.Body).Decode(&loginRequest)

	encryptedBytes, _ := base64.StdEncoding.DecodeString(loginRequest.EncryptedPassword)

	role, authenticated := auth.Authenticate(encryptedBytes)

	if authenticated {
		writer.Header().Add("UUID", fmt.Sprintf("%v", auth.GetUUID(loginRequest.Username)))
		writer.Header().Add("Certificate", fmt.Sprintf("%v", auth.GetCertificate(loginRequest.Username, role)))
	}

	fmt.Fprintf(writer, "User accepted as: %s", role)
}

func servePublicKey(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "application/x-pem-file")

	fmt.Fprintf(writer, "%v", auth.GetPublicKey())
}

func handleSheetChange(writer http.ResponseWriter, request *http.Request) {
	requestBytes, _ := io.ReadAll(request.Body)

	newID := string(requestBytes)

	response := sheet.UpdateSheetID(newID)

	fmt.Fprintf(writer, "%s", response)
}
