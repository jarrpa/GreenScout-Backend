package server

import (
	"GreenScoutBackend/constants"
	"GreenScoutBackend/lib"
	"GreenScoutBackend/rsaUtil"
	"GreenScoutBackend/schedule"
	"GreenScoutBackend/setup"
	"GreenScoutBackend/sheet"
	"GreenScoutBackend/userDB"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	allJson, _ := os.ReadDir(filepath.Join("InputtedJson", "In"))

	// Avoid nil files
	if len(allJson) > 0 {
		// Only deal with first file
		file := allJson[0]

		// Parse and write to spreadsheet
		team, hadErrs := lib.Parse(file.Name(), false)

		if !hadErrs {
			if allMatching := lib.GetAllMatching(file.Name()); constants.CachedConfigs.UsingMultiScouting && len(allMatching) > 0 {
				var entries []lib.TeamData
				entries = append(entries, team)
				for _, foundFile := range allMatching {
					parsedData, foundErrs := lib.Parse(foundFile, true)
					if !foundErrs {
						entries = append(entries, parsedData)
					} else {
						lib.MoveFile(filepath.Join("InputtedJson", "Written", foundFile), filepath.Join("InputtedJson", "Errored"))
					}
				}
				sheet.WriteMultiScoutedTeamDataToLine(
					lib.CompileMultiMatch(entries...),
					lib.GetRow(team),
					entries,
				)
			} else {
				sheet.WriteTeamDataToLine(team, lib.GetRow(team))
			}

			lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Written", file.Name()))
			println("Successfully Processed " + file.Name())
		} else {
			lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
			println("Errors in processing " + filepath.Join("InputtedJson", "In", file.Name()) + ", moved to " + filepath.Join("InputtedJson", "Errored", file.Name()))
		}

	}
}

func SetupServer() *http.Server {
	//No authentication
	http.HandleFunc("/", handleWithCORS(handleRoot, true))
	http.HandleFunc("/pub", handleWithCORS(servePublicKey, false))
	http.HandleFunc("/schedule", handleWithCORS(handleScheduleRequest, true))
	http.HandleFunc("/leaderboard", handleWithCORS(serveLeaderboard, true))
	http.HandleFunc("/scouterLookup", handleWithCORS(serveMatchScouter, true))
	http.HandleFunc("/userInfo", handleWithCORS(serveUserInfo, true))
	http.HandleFunc("/certificateValid", handleWithCORS(handleCertificateVerification, false))

	//Provides Authentication
	http.HandleFunc("/login", handleWithCORS(handleLoginRequest, false))

	//Any Authentication
	http.HandleFunc("/dataEntry", handleWithCORS(postJson, true))
	http.HandleFunc("/singleSchedule", handleWithCORS(serveScouterSchedule, true))

	//Admin or curr user
	http.HandleFunc("/setDisplayName", handleWithCORS(setDisplayName, true))

	//Admin tools
	http.HandleFunc("/addSchedule", handleWithCORS(addIndividualSchedule, true))
	http.HandleFunc("/modScore", handleWithCORS(handleScoreChange, true))
	http.HandleFunc("/allUsers", handleWithCORS(serveUsersRequest, true))
	http.HandleFunc("/addBadge", handleWithCORS(addBadge, true))

	//Super only
	http.HandleFunc("/keyChange", handleWithCORS(handleKeyChange, false))
	http.HandleFunc("/sheetChange", handleWithCORS(handleSheetChange, false))

	jsrv := &http.Server{
		Addr: ":8443",
		// ReadTimeout:  20 * time.Second,
		// WriteTimeout: 20 * time.Second,
	}

	return jsrv
}

func handleRoot(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "howdy!")
}

func postJson(writer http.ResponseWriter, request *http.Request) {

	_, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate")) //Don't care about specific role for post, everyone that is auth'd can.

	if authenticated {
		requestBytes, _ := io.ReadAll(request.Body)

		var team lib.TeamData
		err := json.Unmarshal(requestBytes, &team)

		if err != nil {
			println("Mangle Error: " + err.Error())

			mangledFile, _ := os.Create(filepath.Join("InputtedJson", "Mangled", time.Now().String()+".json"))

			defer mangledFile.Close()

			fmt.Fprint(mangledFile, string(requestBytes))

			writer.WriteHeader(500)
			fmt.Fprint(writer, ":(")
		} else {

			fileName := fmt.Sprintf("%s_%v_%s_%v", lib.GetCurrentEvent(), team.Match.Number, lib.GetDSString(team.DriverStation.IsBlue, uint(team.DriverStation.Number)), time.Now().UnixMilli())
			//EVENT_MATCH_{COLOR}{DSNUM}_SystemTimeMS

			file, err := os.Create(filepath.Join("InputtedJson", "In", fileName+".json"))
			if err != nil {
				panic(err)
			}
			defer file.Close()

			encoder := json.NewEncoder(file)
			encoder.Encode(team)

			if request.Header.Get("joshtown") == "tumble" { //This was used for testing during 2024 GCR. It also used to be more crudely worded.
				writer.WriteHeader(500)
			}

			fmt.Fprintf(writer, "Processed %v\n", fileName)
		}
	} else {
		writer.WriteHeader(500)
		fmt.Fprintf(writer, "Not authenticated :(")
	}
}

func handleKeyChange(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && role == "super" {
		requestBytes, _ := io.ReadAll(request.Body)

		newKey := string(requestBytes)

		if setup.SetEventKey(newKey) {
			fmt.Fprintf(writer, "Successfully changed event key to "+newKey+"\n")
		} else {
			fmt.Fprintf(writer, "There was a problem changing the event key to "+newKey+", make sure it's valid!\n")
		}
	} else if !authenticated {
		fmt.Fprintf(writer, "Not successfully authenticated. Please ensure you have correct login details.\n")
	} else {
		fmt.Fprintf(writer, "Not a super user. womp womp\n")
	}
}

func handleScheduleRequest(writer http.ResponseWriter, request *http.Request) {
	file, _ := os.Open(filepath.Join("schedule", "schedule.json"))

	fileBytes, _ := io.ReadAll(file)

	fmt.Fprintf(writer, "%s", string(fileBytes))
}

func handleLoginRequest(writer http.ResponseWriter, request *http.Request) {
	var loginRequest userDB.LoginAttempt

	json.NewDecoder(request.Body).Decode(&loginRequest)

	encryptedBytes, _ := base64.StdEncoding.DecodeString(loginRequest.EncryptedPassword)

	role, authenticated := userDB.Authenticate(encryptedBytes)

	if authenticated {
		writer.Header().Add("UUID", fmt.Sprintf("%v", userDB.GetUUID(loginRequest.Username)))
		writer.Header().Add("Certificate", fmt.Sprintf("%v", userDB.GetCertificate(loginRequest.Username, role)))
	}

	writer.Header().Add("Role", role)

	fmt.Fprintf(writer, "User accepted as: %s", role)
}

func servePublicKey(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "application/x-pem-file")

	fmt.Fprintf(writer, "%v", rsaUtil.GetPublicKey())
}

func handleSheetChange(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		requestBytes, _ := io.ReadAll(request.Body)

		newID := string(requestBytes)

		response := sheet.UpdateSheetID(newID)

		fmt.Fprintf(writer, "%s", response)
	}
}

func serveScouterSchedule(writer http.ResponseWriter, request *http.Request) {
	requestBytes, _ := io.ReadAll(request.Body)

	nameToLookup := string(requestBytes)

	response := schedule.RetrieveSingleScouter(nameToLookup, false)

	fmt.Fprintf(writer, "%s", response)

}

func addIndividualSchedule(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		requestBytes, _ := io.ReadAll(request.Body)
		var requestStruct schedule.ScoutRanges

		nameToLookup := request.Header.Get("userInput")
		json.Unmarshal(requestBytes, &requestStruct)

		schedule.AddIndividualSchedule(nameToLookup, true, requestStruct)

		fmt.Fprintf(writer, "Successfully added schedule for "+nameToLookup)
	}
}

func serveLeaderboard(writer http.ResponseWriter, request *http.Request) {
	json.NewEncoder(writer).Encode(userDB.GetLeaderboard())
}

func handleScoreChange(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		requestBytes, _ := io.ReadAll(request.Body)
		var requestStruct userDB.ModRequest

		json.Unmarshal(requestBytes, &requestStruct)

		userDB.ModifyUserScore(requestStruct.Name, requestStruct.Mod, requestStruct.By)

		fmt.Fprintf(writer, "Successfuly modified score of %s", requestStruct.Name)
	}
}

func handleWithCORS(handler http.HandlerFunc, okCode bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://thegreenmachine.github.io")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*, Certificate")
		w.Header().Set("Access-Control-Expose-Headers", "*, Certificate")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if okCode {
			w.WriteHeader(200)
		}
		handler(w, r)
	}
}

func serveUsersRequest(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		json.NewEncoder(writer).Encode(userDB.GetAllUsers())
	}
}

func serveMatchScouter(writer http.ResponseWriter, request *http.Request) {

	var match lib.MatchInfoRequest
	json.NewDecoder(request.Body).Decode(&match)

	fmt.Fprintf(writer, "%s", lib.GetNameFromWritten(match))
}

func serveUserInfo(writer http.ResponseWriter, request *http.Request) {
	info := userDB.GetUserInfo(request.Header.Get("username"))
	json.NewEncoder(writer).Encode(info)
}

func setDisplayName(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	isUser := userDB.GetUUID(request.Header.Get("username")) == request.Header.Get("uuid")

	if (authenticated && (role == "admin" || role == "super")) || isUser {
		userDB.SetDisplayName(request.Header.Get("username"), request.Header.Get("displayName"))

		info := userDB.GetUserInfo(request.Header.Get("username"))
		json.NewEncoder(writer).Encode(info)
	}
}

func addBadge(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		usernameToAdd := request.Header.Get("username")
		uuid := userDB.GetUUID(usernameToAdd)

		var badge userDB.Badge
		json.NewDecoder(request.Body).Decode(&badge)
		userDB.AddBadge(uuid, badge)

		fmt.Fprintf(writer, "Successfully added %s to %s", badge.ID, usernameToAdd)
	}
}

func handleCertificateVerification(writer http.ResponseWriter, request *http.Request) { //TODO write this method lole
	_, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))

	// println(request.Header.Get("Certificate"))
	// println(authenticated)
	if !authenticated {
		// writer.WriteHeader(500)
	}
}
