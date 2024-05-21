package server

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
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
	allJson, readErr := os.ReadDir(filepath.Join("InputtedJson", "In"))
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading file %v", filepath.Join("InputtedJson", "In"))
		return
	}

	// Avoid nil files
	if len(allJson) > 0 {
		// Only deal with first file
		file := allJson[0]

		// Parse and write to spreadsheet
		team, hadErrs := lib.Parse(file.Name(), false)

		var successfullyWrote bool

		if !hadErrs {
			if allMatching := lib.GetAllMatching(file.Name()); constants.CachedConfigs.UsingMultiScouting && len(allMatching) > 0 {
				var entries []lib.TeamData
				entries = append(entries, team)
				for _, foundFile := range allMatching {
					parsedData, foundErrs := lib.Parse(foundFile, true)
					if !foundErrs {
						entries = append(entries, parsedData)
					} else {
						if !lib.MoveFile(filepath.Join("InputtedJson", "Written", foundFile), filepath.Join("InputtedJson", "Errored", foundFile)) {
							greenlogger.FatalLogMessage("File " + filepath.Join("InputtedJson", "Written", foundFile) + " unable to be moved to Errored, investigate this!")
							//Slack integration - notification
						} else {
							println("Errors in processing " + filepath.Join("InputtedJson", "Written", foundFile) + ", moved to " + filepath.Join("InputtedJson", "Errored", foundFile))
						}
					}
				}
				successfullyWrote = sheet.WriteMultiScoutedTeamDataToLine(
					lib.CompileMultiMatch(entries...),
					lib.GetRow(team),
					entries,
				)
			} else {
				successfullyWrote = sheet.WriteTeamDataToLine(team, lib.GetRow(team))
			}

			//Currently, there is no handling if one can't move. It will loop infinitley.
			if successfullyWrote {
				lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Written", file.Name()))
				greenlogger.LogMessagef("Successfully Processed %v ", file.Name())
			} else {
				lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
				greenlogger.LogMessagef("Errors in writing %v to sheet, moved to %v", filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
			}
		} else {
			lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
			greenlogger.LogMessagef("Errors in processing %v, moved to %v", filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
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
	httpResponsef(writer, "Problem writing http response to root request", "howdy!")
}

func postJson(writer http.ResponseWriter, request *http.Request) {

	_, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate")) //Don't care about specific role for post, everyone that is auth'd can.

	if authenticated {
		requestBytes, readErr := io.ReadAll(request.Body)

		if readErr != nil {
			greenlogger.LogErrorf(readErr, "Problem reading %v", request.Body)
		}

		var team lib.TeamData
		unmarshalErr := json.Unmarshal(requestBytes, &team)

		if unmarshalErr != nil {
			greenlogger.LogErrorf(unmarshalErr, "MANGLED: %v", requestBytes)

			newFileName := filepath.Join("InputtedJson", "Mangled", time.Now().String()+".json")
			mangledFile, createErr := os.Create(newFileName)
			if createErr != nil {
				greenlogger.LogErrorf(createErr, "Problem creating %v", newFileName)
			}

			defer mangledFile.Close()

			writer.WriteHeader(500)

			httpResponsef(writer, "Problem writing http response to Mangled JSON", ":(")
		} else {
			//EVENT_MATCH_{COLOR}{DSNUM}_SystemTimeMS
			fileName := fmt.Sprintf(
				"%s_%v_%s_%v",
				lib.GetCurrentEvent(),
				team.Match.Number,
				lib.GetDSString(team.DriverStation.IsBlue, uint(team.DriverStation.Number)),
				time.Now().UnixMilli(),
			)

			file, createErr := os.Create(filepath.Join("InputtedJson", "In", fileName+".json"))
			if createErr != nil {
				greenlogger.LogErrorf(createErr, "Problem creating %v", filepath.Join("InputtedJson", "In", fileName+".json"))
			}
			defer file.Close()

			encodeErr := json.NewEncoder(file).Encode(&team)
			if encodeErr != nil {
				greenlogger.LogErrorf(encodeErr, "Problem encoding %v", team)
			}

			if request.Header.Get("joshtown") == "tumble" { //This was used for testing during 2024 GCR. It also used to be more crudely worded.
				writer.WriteHeader(500)
			}

			httpResponsef(writer, "Problem writing http response to JSON post request", "Processed %v\n", fileName)
		}
	} else {
		writer.WriteHeader(500)
		httpResponsef(writer, "Problem writing http response to JSON post request with insufficient authentication", "Not authenticated :(")
	}
}

func handleKeyChange(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && role == "super" {
		requestBytes, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			greenlogger.LogErrorf(readErr, "Problem reading %v", request.Body)
			return
		}

		newKey := string(requestBytes)

		if setup.SetEventKey(newKey) {
			httpResponsef(writer, "Problem writing http response to successful event key change", "Successfully changed event key to %v\n", newKey)
		} else {
			httpResponsef(writer, "Problem writing http response to unsuccessful event key change", "There was a problem changing the event key to %v, make sure it's valid!\n", newKey)
		}
	} else if !authenticated {
		httpResponsef(writer, "Problem writing http response to unauthorized attempt to change event key", "Not successfully authenticated. Please ensure you have correct login details.\n")
	} else {
		httpResponsef(writer, "Problem writing http response to non-super attempt to change event key", "Not a super user. womp womp\n")
	}
}

func handleScheduleRequest(writer http.ResponseWriter, request *http.Request) {
	file, openErr := os.Open(filepath.Join("schedule", "schedule.json"))
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem opening %v", filepath.Join("schedule", "schedule.json"))
	}

	fileBytes, readErr := io.ReadAll(file)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", request.Body)
	}

	httpResponsef(writer, "Problem writing http response to schedule request", "%s", string(fileBytes))
}

func handleLoginRequest(writer http.ResponseWriter, request *http.Request) {
	var loginRequest userDB.LoginAttempt

	decodeErr := json.NewDecoder(request.Body).Decode(&loginRequest)
	if decodeErr != nil {
		greenlogger.LogErrorf(decodeErr, "Problem decoding %v", request.Body)
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(loginRequest.EncryptedPassword)
	if err != nil {
		greenlogger.LogErrorf(err, "Problem decoding %v", loginRequest.EncryptedPassword)
	}

	role, authenticated := userDB.Authenticate(encryptedBytes)

	if authenticated {
		writer.Header().Add("UUID", fmt.Sprintf("%v", userDB.GetUUID(loginRequest.Username)))
		writer.Header().Add("Certificate", fmt.Sprintf("%v", userDB.GetCertificate(loginRequest.Username, role)))
	}

	writer.Header().Add("Role", role)

	httpResponsef(writer, "Problem writing http response to login request", "User accepted as: %s", role)
}

func servePublicKey(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "application/x-pem-file")

	httpResponsef(writer, "Problem serving public key", "%v", rsaUtil.GetPublicKey())
}

func handleSheetChange(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		requestBytes, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			greenlogger.LogErrorf(readErr, "Problem reading %v", request.Body)
		}

		newID := string(requestBytes)

		response := sheet.UpdateSheetID(newID)

		httpResponsef(writer, "Problem writing http response to sheet change request", "%s", response)
	}
}

func serveScouterSchedule(writer http.ResponseWriter, request *http.Request) {
	requestBytes, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", request.Body)
	}

	nameToLookup := string(requestBytes)

	response := schedule.RetrieveSingleScouter(nameToLookup, false)

	httpResponsef(writer, "Problem serving scouter schedule", "%s", response)
}

func addIndividualSchedule(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		requestBytes, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			greenlogger.LogErrorf(readErr, "Problem reading %v", request.Body)
		}
		var requestStruct schedule.ScoutRanges

		nameToLookup := request.Header.Get("userInput")
		unmarshalErr := json.Unmarshal(requestBytes, &requestStruct)
		if unmarshalErr != nil {
			greenlogger.LogErrorf(unmarshalErr, "Error unmarshalling %v", requestBytes)
		}

		schedule.AddIndividualSchedule(nameToLookup, true, requestStruct)

		httpResponsef(writer, "Problem writing http response for individual schedule change request", "Successfully added schedule for %s", nameToLookup)
	}
}

func serveLeaderboard(writer http.ResponseWriter, request *http.Request) {
	leaderboard := userDB.GetLeaderboard()
	encodeErr := json.NewEncoder(writer).Encode(userDB.GetLeaderboard())
	if encodeErr != nil {
		greenlogger.LogErrorf(encodeErr, "Problem encoding %v", leaderboard)
	}
}

func handleScoreChange(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		requestBytes, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			greenlogger.LogErrorf(readErr, "Problem reading %v", request.Body)
		}

		var requestStruct userDB.ModRequest

		unmarshalErr := json.Unmarshal(requestBytes, &requestStruct)
		if unmarshalErr != nil {
			greenlogger.LogErrorf(unmarshalErr, "Error unmarshalling %v", requestBytes)
		}

		userDB.ModifyUserScore(requestStruct.Name, requestStruct.Mod, requestStruct.By)

		httpResponsef(writer, "Problem writing http response for score change request", "Successfully modified score of %s", requestStruct.Name)
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
		users := userDB.GetAllUsers()
		encodeErr := json.NewEncoder(writer).Encode(userDB.GetAllUsers())
		if encodeErr != nil {
			greenlogger.LogErrorf(encodeErr, "Problem encoding %v", users)
		}
	}
}

func serveMatchScouter(writer http.ResponseWriter, request *http.Request) {

	var match lib.MatchInfoRequest
	decodeErr := json.NewDecoder(request.Body).Decode(&match)
	if decodeErr != nil {
		greenlogger.LogErrorf(decodeErr, "Problem decoding %v", request.Body)
	}

	httpResponsef(writer, "Problem serving scouter for a given match", "%s", lib.GetNameFromWritten(match))
}

func serveUserInfo(writer http.ResponseWriter, request *http.Request) {
	info := userDB.GetUserInfo(request.Header.Get("username"))
	encodeErr := json.NewEncoder(writer).Encode(info)
	if encodeErr != nil {
		greenlogger.LogErrorf(encodeErr, "Problem encoding %v", info)
	}

}

func setDisplayName(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	isUser := userDB.GetUUID(request.Header.Get("username")) == request.Header.Get("uuid")

	if (authenticated && (role == "admin" || role == "super")) || isUser {
		userDB.SetDisplayName(request.Header.Get("username"), request.Header.Get("displayName"))

		info := userDB.GetUserInfo(request.Header.Get("username"))
		encodeErr := json.NewEncoder(writer).Encode(info)
		if encodeErr != nil {
			greenlogger.LogErrorf(encodeErr, "Problem encoding %v", info)
		}

	}
}

func addBadge(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		usernameToAdd := request.Header.Get("username")
		uuid := userDB.GetUUID(usernameToAdd)

		var badge userDB.Badge
		decodeErr := json.NewDecoder(request.Body).Decode(&badge)
		if decodeErr != nil {
			greenlogger.LogErrorf(decodeErr, "Problem decoding %v", request.Body)
		}

		userDB.AddBadge(uuid, badge)

		httpResponsef(writer, "Problem writing http response for badge addition request", "Successfully added %s to %s", badge.ID, usernameToAdd)
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

func httpResponsef(writer http.ResponseWriter, errDescription string, message string, args ...any) {
	_, err := fmt.Fprintf(writer, message, args...)

	if err != nil {
		greenlogger.LogError(err, errDescription)
	}
}
