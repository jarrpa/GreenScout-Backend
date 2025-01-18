package sheet

// Utilites for accessing the google sheets API

import (
	"GreenScoutBackend/constants"
	filemanager "GreenScoutBackend/fileManager"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/lib"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

// Early methods (setup) are from google's quickstart, so I didn't change much about them

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := constants.SheetsTokenFile
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	greenlogger.LogMessagef("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		greenlogger.FatalError(err, "Unable to read authorization code: ")
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		greenlogger.FatalError(err, "Unable to retrieve token from web: ")
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	greenlogger.LogMessagef("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		greenlogger.FatalError(err, "Unable to cache oauth token: ")
	}
	defer f.Close()
	encodeErr := json.NewEncoder(f).Encode(token)
	if encodeErr != nil {
		greenlogger.FatalError(encodeErr, "Unable to encode token to file")
	}
}

// The spreadsheet ID, held in memory
var SpreadsheetId string

// The service (api instance), held in memory
var Srv *sheets.Service

// Sets up the sheets API based on the credentials.json and token.json
func SetupSheetsAPI(b []byte) {
	ctx := context.Background()

	// If modifying these scopes, delete your previously saved token.json.
	//config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	client, err := google.DefaultClient(context.Background(), sheets.SpreadsheetsScope)
	if err != nil {
		greenlogger.FatalError(err, "Unable to parse client secret file to config: %v")
	}
	//client := getClient(config)

	Srv, err = sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		greenlogger.FatalError(err, "Unable to retrieve Sheets client: %v")
	}
}

// Writes team data from multi-scouting to a specified line
func WriteMultiScoutedTeamDataToLine(matchdata lib.MultiMatch, row int, sources []lib.TeamData) bool {
	ampTendency, speakerTendency, distanceTendency, shuttleTendency := lib.GetCycleTendencies(matchdata.CycleData.AllCycles)
	ampAccuracy, speakerAccuracy, distanceAccuracy, shuttleAccuracy := lib.GetCycleAccuracies(matchdata.CycleData.AllCycles)

	// This is ONE ROW. Each value is a cell in that row.
	valuesToWrite := []interface{}{
		matchdata.TeamNumber,
		matchdata.CycleData.AvgCycleTime,
		matchdata.CycleData.NumCycles,
		math.Round(ampTendency*10000) / 100,                   // Amp tendency
		ampAccuracy,                                           // Amp Accuracy
		math.Round(speakerTendency*10000) / 100,               // Speaker tendency
		speakerAccuracy,                                       // Speaker Accuracy
		math.Round(distanceTendency*10000) / 100,              // Distance tendency
		distanceAccuracy,                                      // Distance accuracy
		math.Round(shuttleTendency*10000) / 100,               // Shuttle tendency
		shuttleAccuracy,                                       // Shuttle accuracy
		lib.GetSpeakerPosAsString(matchdata.SpeakerPositions), // Speaker positions
		lib.GetPickupLocations(matchdata.Pickups),             // Pickup positions
		matchdata.Auto.Can,                                    // Had Auto
		matchdata.Auto.Scores,                                 // Scores in auto
		lib.GetAutoAccuracy(matchdata.Auto),                   // Auto accuracy
		matchdata.Auto.Ejects,                                 // Auto shuttles
		matchdata.Climb.Succeeded,                             // Can climb
		matchdata.Climb.Time,                                  // Climb Time
		matchdata.Parked,                                      // Parked
		matchdata.TrapScore,                                   // Trap Score
		lib.CompileNotes2(matchdata, sources),                 // Notes + Penalties + DC + Lost track
	}

	var vr sheets.ValueRange

	vr.Values = append(vr.Values, valuesToWrite)

	writeRange := fmt.Sprintf("RawData!B%v", row)

	_, err := Srv.Spreadsheets.Values.Update(SpreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()

	if err != nil {
		greenlogger.LogError(err, "Unable to write data to sheet")
		return false
	}
	return true
}

// Writes data from a single-scouted match to a line
func WriteTeamDataToLine(teamData lib.TeamData, row int) bool {
	ampTendency, speakerTendency, distanceTendency, shuttleTendency := lib.GetCycleTendencies(teamData.Cycles)
	ampAccuracy, speakerAccuracy, distanceAccuracy, shuttleAccuracy := lib.GetCycleAccuracies(teamData.Cycles)

	// This is ONE ROW. Each value is a cell in that row.
	valuesToWrite := []interface{}{
		teamData.TeamNumber,                           // Team Number
		lib.GetAvgCycleTime(teamData.Cycles),          // Avg cycle time
		lib.GetNumCycles(teamData.Cycles),             // Num Cycles
		math.Round(ampTendency*10000) / 100,           // Amp tendency
		ampAccuracy,                                   // Amp Accuracy
		math.Round(speakerTendency*10000) / 100,       // Speaker tendency
		speakerAccuracy,                               // Speaker Accuracy
		math.Round(distanceTendency*10000) / 100,      // Distance tendency
		distanceAccuracy,                              // Distance accuracy
		math.Round(shuttleTendency*10000) / 100,       // Shuttle tendency
		shuttleAccuracy,                               // Shuttle accuracy
		lib.GetSpeakerPosAsString(teamData.Positions), // Speaker positions
		lib.GetPickupLocations(teamData.Pickups),      // Pickup positions
		teamData.Auto.Can,                             // Had Auto
		teamData.Auto.Scores,                          // Scores in auto
		lib.GetAutoAccuracy(teamData.Auto),            // Auto accuracy
		teamData.Auto.Ejects,                          // Auto shuttles
		teamData.Climb.Succeeded,                      // Can climb
		teamData.Climb.Time,                           // Climb Time
		teamData.Misc.Parked,                          // Parked
		teamData.Trap.Score,                           // Trap Score
		lib.CompileNotes(teamData),                    // Notes + Penalties + DC + Lost track
	}

	var vr sheets.ValueRange

	vr.Values = append(vr.Values, valuesToWrite)

	writeRange := fmt.Sprintf("RawData!B%v", row)

	_, err := Srv.Spreadsheets.Values.Update(SpreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()

	if err != nil {
		greenlogger.LogError(err, "Unable to write data to sheet")
		return false
	}

	return true

}

// Wrapper around sheets' batch update.
func BatchUpdate(dataset [][]interface{}, writeRange string) {
	rb := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
	}

	rb.Data = append(rb.Data, &sheets.ValueRange{
		Range:  writeRange,
		Values: dataset,
	})

	_, err := Srv.Spreadsheets.Values.BatchUpdate(SpreadsheetId, rb).Do()

	if err != nil {
		greenlogger.LogError(err, "Unable to write data to sheet")
	}
}

// Fills sheet with all matches from that event.
func FillMatches(startMatch int, endMatch int) {
	if !(math.Abs(float64(endMatch)-float64(startMatch)) >= 50) {

		matchTracker := 2 + (startMatch-1)*6

		for i := startMatch; i <= endMatch; i++ {

			perMatchInterface := [][]interface{}{ // 6 numbers, all same
				{i}, {i}, {i}, {i}, {i}, {i},
			}

			BatchUpdate(perMatchInterface, fmt.Sprintf("RawData!A%v:A%v", matchTracker, matchTracker+6))
			matchTracker += 6
		}
	} else {
		greenlogger.LogMessage("Input matches with a delta under 50!")
	}
}

// Updates the ID of the sheet to be used, in memory and yaml.
func UpdateSheetID(newSheet string) string {
	if IsSheetValid(newSheet) {
		constants.CachedConfigs.SpreadSheetID = newSheet

		configFile, openErr := filemanager.OpenWithPermissions(constants.ConfigFilePath)
		if openErr != nil {
			greenlogger.LogErrorf(openErr, "Problem opening %v", constants.ConfigFilePath)
			return "There was a problem updating the sheet ID"
		}

		defer configFile.Close()

		encodeErr := yaml.NewEncoder(configFile).Encode(&constants.CachedConfigs)

		if encodeErr != nil {
			greenlogger.LogErrorf(encodeErr, "Problem encoding %v", constants.CachedConfigs)
			return "There was a problem updating the sheet ID"
		}

		return "Successfully updated sheet ID to " + newSheet
	}
	return "Sheet ID " + newSheet + " is invalid!"

}

// Tries to read the top-left cell of the RawData tab, returning if it can.
func IsSheetValid(id string) bool {
	spreadsheetId := id
	readRange := "RawData!A1:1"
	_, err := Srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	return err == nil
}

// Adds conditinoal formatting to the raw data tab.
// This consists of two sinusoidal functions that ensure 3-red 3-blue coloring.
func WriteConditionalFormatting() {

	tabs, _ := Srv.Spreadsheets.Get(SpreadsheetId).Do()

	var sheetID int64

	for _, sheet := range tabs.Sheets {
		if sheet.Properties.Title == "RawData" {
			sheetID = sheet.Properties.SheetId
			break
		}
	}

	_, sheetErr := Srv.Spreadsheets.BatchUpdate(
		SpreadsheetId,
		&sheets.BatchUpdateSpreadsheetRequest{

			Requests: []*sheets.Request{
				{
					AddConditionalFormatRule: &sheets.AddConditionalFormatRuleRequest{
						Index: 0,
						Rule: &sheets.ConditionalFormatRule{
							BooleanRule: &sheets.BooleanRule{
								Condition: &sheets.BooleanCondition{
									Type: "CUSTOM_FORMULA",
									Values: []*sheets.ConditionValue{
										{UserEnteredValue: "=(SIN(((PI() /3)) * (ROW()-1) -0.5)) > 0"},
									},
								},
								Format: &sheets.CellFormat{
									BackgroundColor: &sheets.Color{
										Red:   1,
										Alpha: 1, // https://steamuserimages-a.akamaihd.net/ugc/2040738890178501955/DB9342C662AFAF139B605B3B6EBF593ADF42550E/?imw=637&imh=358&ima=fit&impolicy=Letterbox&imcolor=%23000000
									},
								},
							},
							Ranges: []*sheets.GridRange{
								{
									SheetId:          sheetID,
									StartRowIndex:    1,
									StartColumnIndex: 0,
									EndColumnIndex:   1,
								},
							},
						},
					},
				},
				{
					AddConditionalFormatRule: &sheets.AddConditionalFormatRuleRequest{
						Index: 1,
						Rule: &sheets.ConditionalFormatRule{
							BooleanRule: &sheets.BooleanRule{
								Condition: &sheets.BooleanCondition{
									Type: "CUSTOM_FORMULA",
									Values: []*sheets.ConditionValue{
										{UserEnteredValue: "=(SIN(((PI() /3)) * (ROW()-1) -0.5)) < 0"},
									},
								},
								Format: &sheets.CellFormat{
									BackgroundColor: &sheets.Color{
										Red:   164.0 / 255.0,
										Green: 194.0 / 255.0,
										Blue:  244.0 / 255.0,
										Alpha: 1, // https://i1.sndcdn.com/artworks-JyCZdFbdVSMdUUjr-driMCA-t500x500.jpg
									},
								},
							},
							Ranges: []*sheets.GridRange{
								{
									SheetId:          sheetID,
									StartRowIndex:    1,
									StartColumnIndex: 0,
									EndColumnIndex:   1,
								},
							},
						},
					},
				},
			},
		},
	).Do()

	if sheetErr != nil {
		greenlogger.LogError(sheetErr, "Problem adding conditional formatting.")
	}
}

// Writes data from pit scouting to a line
func WritePitDataToLine(pitData lib.PitScoutingData, row int) bool {

	// This is ONE ROW. Each value is a cell in that row.
	valuesToWrite := []interface{}{
		pitData.TeamNumber,                       // Team Number
		pitData.PitIdentifier,                    // Pit Identifier
		pitData.Drivetrain,                       // Drivetrain type
		lib.GetSpeakerPosAsString(pitData.Sides), // Speaker positions
		pitData.Distance.Can,                     // Can distance at all
		lib.GetDistance(pitData),                 // Distance from which shooting is possible
		pitData.AutoScores,                       // Auto Scores
		pitData.MiddleControls,                   // Middle notes controllable in auto
		pitData.NoteDetection,                    // Has note detection
		pitData.Cycles,                           // Avg cycles
		pitData.DriverExperience,                 // Driver years of experience
		pitData.BotType,                          // ex. Amp, Speaker, Defense
		pitData.EndgameBehavior,                  // Climb or park basically
		lib.GetClimbTime(pitData),                // Time to climb
	}

	var vr sheets.ValueRange

	vr.Values = append(vr.Values, valuesToWrite)

	writeRange := fmt.Sprintf("PitScouting!B%v", row)

	_, err := Srv.Spreadsheets.Values.Update(SpreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()

	if err != nil {
		greenlogger.LogError(err, "Unable to write data to sheet")
		return false
	}

	return true

}
