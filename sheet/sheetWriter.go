package sheet

import (
	"GreenScoutBackend/lib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
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
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
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
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

var spreadsheetId string
var srv *sheets.Service

func SetupSheetsAPI() {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err = sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	spreadsheetId = retrieveSheetID()
}

func WriteTeamDataToLine(teamData lib.TeamData, row int) {
	cycleNums := lib.GetCyclePercents(teamData)

	valuesToWrite := []interface{}{
		teamData.TeamNumber,                           // Team num in case of wackyness/replacement
		lib.GetAvgCycleTime(teamData.Cycles),          // Avg  Cycle time
		lib.GetNumCycles(teamData.Cycles),             // Number of Cycles
		math.Round(cycleNums[0]*10000) / 100,          // % Of amp cycles
		math.Round(cycleNums[1]*10000) / 100,          // % Of speaker cycles
		lib.GetSpeakerPosAsString(teamData.Positions), // Speakers they did @ least once
		teamData.DistanceShooting.Can,                 // Did they distance shoot
		lib.GetAccuracy(teamData.DistanceShooting),    // How Accurate was their distance shooting (subjective) - just have people enter in whole numbers
		teamData.Auto.Can,                             // Did they have any auto
		teamData.Auto.Succeeded,                       // Did their auto succeed in its goals
		teamData.Auto.Scores,                          // How many times did they score in auto
		teamData.Auto.Misses,                          // How many times did they miss in auto
		teamData.Auto.Ejects,                          // How many ejects did they do in auto
		teamData.Climb.Can,                            // Did they try to climb
		teamData.Climb.Time,                           // Time to climb
		teamData.Misc.Parked,                          // Did they park
		teamData.Misc.DC,                              // Did the robot DC
		teamData.Misc.LostTrack,                       // Did the scouter lose track
		teamData.Trap.Score,                           // Trap score
		//If people want, add trap accuracy
		strings.Join(teamData.Penalties, "; "), // Penalties
		teamData.Notes,                         // Notes
	}
	var vr sheets.ValueRange

	vr.Values = append(vr.Values, valuesToWrite)

	writeRange := fmt.Sprintf("RawData!B%v", row)

	_, err := srv.Spreadsheets.Values.Update(spreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()

	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}
}

func BatchUpdate(dataset [][]interface{}, writeRange string) {
	rb := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
	}

	rb.Data = append(rb.Data, &sheets.ValueRange{
		Range:  writeRange,
		Values: dataset,
	})

	_, err := srv.Spreadsheets.Values.BatchUpdate(spreadsheetId, rb).Do()

	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}
}

func fillMatches(startMatch int, endMatch int) {
	if !(math.Abs(float64(endMatch)-float64(startMatch)) >= 50) {

		matchTracker := 2 + (startMatch-1)*6

		for i := startMatch; i <= endMatch; i++ {

			perMatchInterface := [][]interface{}{
				{i}, {i}, {i}, {i}, {i}, {i},
			}

			BatchUpdate(perMatchInterface, fmt.Sprintf("RawData!A%v:A%v", matchTracker, matchTracker+6))
			matchTracker += 6
		}
	} else {
		println("Input matches with a delta under 50!")
	}
}

func FillOutTeamSheet(tab string) {
	writeRange := fmt.Sprintf("%s!%s", tab, lib.GetFullTeamRange())
	BatchUpdate(
		lib.GetTeamListAsInterface(),
		writeRange,
	)
}

func retrieveSheetID() string {
	file, _ := os.Open(filepath.Join("sheet", "spreadsheet.txt"))
	defer file.Close()

	dataBytes, _ := io.ReadAll(file)

	return string(dataBytes)
}

func UpdateSheetID(newSheet string) string {
	file, _ := os.Create(filepath.Join("sheet", "spreadsheet.txt"))
	defer file.Close()

	file.WriteString(newSheet)
	return "Successfully updated sheet ID to " + newSheet
}
