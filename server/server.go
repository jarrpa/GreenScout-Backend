package server

import (
	"GreenScoutBackend/constants"
	filemanager "GreenScoutBackend/fileManager"
	"GreenScoutBackend/gallery"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/lib"
	"GreenScoutBackend/pfp"
	"GreenScoutBackend/rsaUtil"
	"GreenScoutBackend/schedule"
	"GreenScoutBackend/setup"
	"GreenScoutBackend/sheet"
	"GreenScoutBackend/userDB"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		if len(strings.Split(file.Name(), "_")) == 2 { // Pit Scouting
			pit, hadErrs := lib.ParsePitScout(file.Name())

			if !hadErrs {
				if sheet.WritePitDataToLine(pit, lib.GetPitRow(pit.TeamNumber)) {
					lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "PitWritten", file.Name()))
					greenlogger.LogMessagef("Successfully Processed %v ", file.Name())
					userDB.ModifyUserScore(pit.Scouter, userDB.Increase, 1)
				} else {
					lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
					greenlogger.LogMessagef("Errors in writing %v to sheet, moved to %v", filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
				}
			} else {
				lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
				greenlogger.LogMessagef("Errors in processing %v, moved to %v", filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Errored", file.Name()))
			}
		} else {
			team, hadErrs := lib.Parse(file.Name(), false)

			var successfullyWrote bool

			if !hadErrs {
				if allMatching := lib.GetAllMatching(file.Name()); constants.CachedConfigs.UsingMultiScouting && len(allMatching) > 0 {
					var entries []lib.TeamData
					entries = append(entries, team)
					for _, foundFile := range allMatching {
						if team.Rescouting {
							if !lib.MoveFile(filepath.Join("InputtedJson", "Written", foundFile), filepath.Join("InputtedJson", "Discarded", foundFile)) {
								greenlogger.LogMessage("File " + filepath.Join("InputtedJson", "Written", foundFile) + " unable to be moved to Discarded")
							}
						} else {
							parsedData, foundErrs := lib.Parse(foundFile, true)
							if !foundErrs {
								entries = append(entries, parsedData)
							} else {
								if !lib.MoveFile(filepath.Join("InputtedJson", "Written", foundFile), filepath.Join("InputtedJson", "Errored", foundFile)) {
									greenlogger.FatalLogMessage("File " + filepath.Join("InputtedJson", "Written", foundFile) + " unable to be moved to Errored, investigate this!")
								} else {
									greenlogger.NotifyMessage("Errors in processing " + filepath.Join("InputtedJson", "Written", foundFile) + ", moved to " + filepath.Join("InputtedJson", "Errored", foundFile))
								}
							}
						}
					}

					if team.Rescouting {
						successfullyWrote = sheet.WriteTeamDataToLine(team, lib.GetRow(team))
					} else {
						successfullyWrote = sheet.WriteMultiScoutedTeamDataToLine(
							lib.CompileMultiMatch(entries...),
							lib.GetRow(team),
							entries,
						)
					}
				} else {
					successfullyWrote = sheet.WriteTeamDataToLine(team, lib.GetRow(team))
				}

				//Currently, there is no handling if one can't move. It will loop infinitley.
				if successfullyWrote {
					lib.MoveFile(filepath.Join("InputtedJson", "In", file.Name()), filepath.Join("InputtedJson", "Written", file.Name()))
					greenlogger.LogMessagef("Successfully Processed %v ", file.Name())
					userDB.ModifyUserScore(team.Scouter, userDB.Increase, 1)
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
	http.HandleFunc("/getPfp", handleWithCORS(handlePfpRequest, true))
	http.HandleFunc("/generalInfo", handleWithCORS(handleGeneralInfoRequest, true))
	http.HandleFunc("/allEvents", handleWithCORS(handleEventsRequest, true))
	http.HandleFunc("/gallery", handleWithCORS(handleGalleryRequest, true))
	http.HandleFunc("/adminUserInfo", handleWithCORS(serveUserInfoForAdmins, true))

	//Provides Authentication
	http.HandleFunc("/login", handleWithCORS(handleLoginRequest, false))

	//Any Authentication
	http.HandleFunc("/dataEntry", handleWithCORS(postJson, true))
	http.HandleFunc("/pitScout", handleWithCORS(postPitScout, true))
	http.HandleFunc("/singleSchedule", handleWithCORS(serveScouterSchedule, true))

	//Admin or curr user
	http.HandleFunc("/setDisplayName", handleWithCORS(setDisplayName, true))
	http.HandleFunc("/setUserPfp", handleWithCORS(setPfp, true))
	http.HandleFunc("/provideAdditions", handleWithCORS(handleFrontendAdditions, true))
	http.HandleFunc("/setColor", handleWithCORS(handleColorChange, true))

	//Admin or verified
	http.HandleFunc("/spreadsheet", handleWithCORS(serveSpreadsheet, true))

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

	if constants.CachedConfigs.LogConfigs.Logging && constants.CachedConfigs.LogConfigs.LoggingHttp {
		jsrv.ErrorLog = greenlogger.GetLogger()
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
			mangledFile, openErr := filemanager.OpenWithPermissions(newFileName)
			if openErr != nil {
				greenlogger.LogErrorf(openErr, "Problem creating %v", newFileName)
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

			file, openErr := filemanager.OpenWithPermissions(filepath.Join("InputtedJson", "In", fileName+".json"))
			if openErr != nil {
				greenlogger.LogErrorf(openErr, "Problem creating %v", filepath.Join("InputtedJson", "In", fileName+".json"))
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

func postPitScout(writer http.ResponseWriter, request *http.Request) {
	_, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate")) //Don't care about specific role for post, everyone that is auth'd can.

	if authenticated {
		requestBytes, readErr := io.ReadAll(request.Body)

		if readErr != nil {
			greenlogger.LogErrorf(readErr, "Problem reading %v", request.Body)
		}

		var pit lib.PitScoutingData
		unmarshalErr := json.Unmarshal(requestBytes, &pit)

		if unmarshalErr != nil {
			greenlogger.LogErrorf(unmarshalErr, "MANGLED: %v", requestBytes)

			newFileName := filepath.Join("InputtedJson", "Mangled", time.Now().String()+".json")
			mangledFile, openErr := filemanager.OpenWithPermissions(newFileName)
			if openErr != nil {
				greenlogger.LogErrorf(openErr, "Problem creating %v", newFileName)
			}

			defer mangledFile.Close()

			writer.WriteHeader(500)

			httpResponsef(writer, "Problem writing http response to Mangled JSON", ":(")
		} else {
			//EVENT_TEAM.json
			fileName := fmt.Sprintf(
				"%s_%v",
				lib.GetCurrentEvent(),
				pit.TeamNumber,
			)

			file, openErr := filemanager.OpenWithPermissions(filepath.Join("InputtedJson", "In", fileName+".json"))
			if openErr != nil {
				greenlogger.LogErrorf(openErr, "Problem creating %v", filepath.Join("InputtedJson", "In", fileName+".json"))
			}
			defer file.Close()

			encodeErr := json.NewEncoder(file).Encode(&pit)
			if encodeErr != nil {
				greenlogger.LogErrorf(encodeErr, "Problem encoding %v", pit)
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
	if authenticated && (role == "Admin" || role == "super") {
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
	if decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
		greenlogger.LogErrorf(decodeErr, "Problem decoding %v", request.Body)
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(loginRequest.EncryptedPassword)
	if err != nil {
		greenlogger.LogErrorf(err, "Problem decoding %v", loginRequest.EncryptedPassword)
	}

	role, authenticated := userDB.Authenticate(encryptedBytes)

	if authenticated {
		uuid, _ := userDB.GetUUID(loginRequest.Username, true)

		writer.Header().Add("UUID", fmt.Sprintf("%v", uuid))
		writer.Header().Add("Certificate", fmt.Sprintf("%v", userDB.GetCertificate(loginRequest.Username, role)))

		if role == "super" {
			userDB.AddBadge(uuid, userDB.Badge{ID: string(userDB.Admin)})
			userDB.AddBadge(uuid, userDB.Badge{ID: string(userDB.Super)})
		} else if role == "admin" {
			userDB.AddBadge(uuid, userDB.Badge{ID: string(userDB.Admin)})
		}
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
	var lbType string
	var typeHeader string = request.Header.Get("type")
	if typeHeader == "HighScore" {
		lbType = "highscore"
	} else if typeHeader == "LifeScore" {
		lbType = "lifescore"
	} else {
		lbType = "score"
	}

	leaderboard := userDB.GetLeaderboard(lbType)
	encodeErr := json.NewEncoder(writer).Encode(leaderboard)
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

	if request.Header.Get("uuid") != "" && userDB.UUIDToUser(request.Header.Get("uuid")) == request.Header.Get("username") {
		var accoladesNotified []userDB.AccoladeData
		for _, accolade := range info.Accolades {
			accoladesNotified = append(accoladesNotified, userDB.AccoladeData{Accolade: accolade.Accolade, Notified: true})
		}

		userDB.SetAccolades(request.Header.Get("uuid"), accoladesNotified)
	}

	encodeErr := json.NewEncoder(writer).Encode(info)
	if encodeErr != nil {
		greenlogger.LogErrorf(encodeErr, "Problem encoding %v", info)
	}

}

func serveUserInfoForAdmins(writer http.ResponseWriter, request *http.Request) {
	info := userDB.GetAdminUserInfo(request.Header.Get("uuid"))

	encodeErr := json.NewEncoder(writer).Encode(info)
	if encodeErr != nil {
		greenlogger.LogErrorf(encodeErr, "Problem encoding %v", info)
	}
}

func setDisplayName(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	uuid, _ := userDB.GetUUID(request.Header.Get("username"), true)

	isUser := uuid == request.Header.Get("uuid")

	if (authenticated && (role == "admin" || role == "super")) || isUser {
		userDB.SetDisplayName(request.Header.Get("username"), request.Header.Get("displayName"))

		info := userDB.GetUserInfo(request.Header.Get("username"))
		writer.WriteHeader(200)
		encodeErr := json.NewEncoder(writer).Encode(info)
		if encodeErr != nil {
			greenlogger.LogErrorf(encodeErr, "Problem encoding %v", info)
		}
	}
}

func setPfp(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	uuid, _ := userDB.GetUUID(request.Header.Get("username"), true)

	isUser := uuid == request.Header.Get("uuid")

	if (authenticated && (role == "admin" || role == "super")) || isUser {
		userDB.SetPfp(request.Header.Get("username"), request.Header.Get("Filename"))
		requestBytes, err := io.ReadAll(request.Body)
		if err != nil {
			greenlogger.LogErrorf(err, "Problem reading %v", request.Body)
		}
		if pfp.WritePfp(requestBytes, request.Header.Get("Filename")) {
			writer.WriteHeader(200)
		} else {
			writer.WriteHeader(500)
		}
	}
}

func handleFrontendAdditions(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	uuid, _ := userDB.GetUUID(request.Header.Get("username"), true)

	isUser := uuid == request.Header.Get("uuid")

	if (authenticated && (role == "admin" || role == "super")) || isUser {
		var Additions userDB.FrontendAdds
		err := json.NewDecoder(request.Body).Decode(&Additions)
		if err != nil {
			greenlogger.LogErrorf(err, "Problem decoding %v", request.Body)
		}

		userDB.ConsumeFrontendAdditions(Additions, true)
	}
}

func handleColorChange(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	uuid, _ := userDB.GetUUID(request.Header.Get("username"), true)

	isUser := uuid == request.Header.Get("uuid")

	if (authenticated && (role == "admin" || role == "super")) || isUser {
		userDB.SetColor(uuid, parseColor(request.Header.Get("color")))
	}
}

func parseColor(colStr string) userDB.LBColor {
	switch colStr {
	case "green":
		return userDB.Green
	case "gold":
		return userDB.Gold
	}
	return userDB.Default
}

func addBadge(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "admin" || role == "super") {
		usernameToAdd := request.Header.Get("username")
		uuid, _ := userDB.GetUUID(usernameToAdd, true)

		var badge userDB.Badge
		decodeErr := json.NewDecoder(request.Body).Decode(&badge)
		if decodeErr != nil {
			greenlogger.LogErrorf(decodeErr, "Problem decoding %v", request.Body)
		}

		userDB.AddBadge(uuid, badge)

		httpResponsef(writer, "Problem writing http response for badge addition request", "Successfully added %s to %s", badge.ID, usernameToAdd)
	}
}

func handleCertificateVerification(writer http.ResponseWriter, request *http.Request) {
	_, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))

	if authenticated {
		writer.WriteHeader(200)
	} else {
		writer.WriteHeader(500)
	}
}

func handlePfpRequest(writer http.ResponseWriter, request *http.Request) {

	username := request.Header.Get("username")

	if username == "" {
		username = request.URL.Query().Get("username")
	}

	pfpPath := userDB.GetUserInfo(username)

	if pfp.CheckForPfp(pfpPath.Pfp) {
		http.ServeFile(writer, request, filepath.Join("pfp", "pictures", pfpPath.Pfp))
	} else {
		http.ServeFile(writer, request, constants.DefaultPfpPath)
	}
}

func handleGeneralInfoRequest(writer http.ResponseWriter, request *http.Request) {
	httpResponsef(writer, "Problem writing response to general info request", `{"EventKey": "%v", "EventName": "%v"}`, lib.GetCurrentEvent(), constants.CachedConfigs.EventKeyName)
}

func handleEventsRequest(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "events.json")
}

func handleGalleryRequest(writer http.ResponseWriter, request *http.Request) {
	ind, err := strconv.ParseInt(request.URL.Query().Get("index"), 10, 64)
	if err != nil {
		greenlogger.LogMessagef("Problem parsing %v as int", request.URL.Query().Get("index"))
	}

	http.ServeFile(writer, request, gallery.GetImage(int(ind)))

}

func serveSpreadsheet(writer http.ResponseWriter, request *http.Request) {
	role, authenticated := userDB.VerifyCertificate(request.Header.Get("Certificate"))
	if authenticated && (role == "1816" || role == "admin" || role == "super") {
		httpResponsef(writer, "Error serving spreadsheet", "https://docs.google.com/spreadsheets/d/"+constants.CachedConfigs.SpreadSheetID)
	}
}

func httpResponsef(writer http.ResponseWriter, errDescription string, message string, args ...any) {
	_, err := fmt.Fprintf(writer, message, args...)

	if err != nil {
		greenlogger.LogError(err, errDescription)
	}
}
