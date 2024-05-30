package setup

import (
	"GreenScoutBackend/constants"
	filemanager "GreenScoutBackend/fileManager"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/lib"
	"GreenScoutBackend/rsaUtil"
	"GreenScoutBackend/schedule"
	"GreenScoutBackend/sheet"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

var configFilePath = filepath.Join("setup", "greenscout.config.yaml")

func TotalSetup(inTesting bool) {
	greenlogger.LogMessage("Retreiving configs...")
	configs := retrieveGeneralConfigs()
	greenlogger.ELogMessagef("General configs retrieved: %v", configs)

	configs.PathToDatabases = "GreenScout-Databases" //this is the only one i'm not having the user enter mainly because git cloning is uniform
	ensureDatabasesExist()
	greenlogger.LogMessage("Essential databases verified...")

	greenlogger.LogMessage("Ensuring sheets API...")
	ensureSheetsAPI()
	greenlogger.ELogMessage("Sheets API confirmed set-up")

	greenlogger.LogMessage("Ensuring sqlite3 driver...")
	configs.SqliteDriver = ensureSqliteDriver()
	greenlogger.ELogMessagef("Sqlite driver validated: %v", configs.SqliteDriver)

	greenlogger.LogMessage("Ensuring InputtedJSON...")
	ensureInputtedJSON()
	greenlogger.ELogMessage("InputtedJSON folders confirmed to exist")

	greenlogger.LogMessage("Ensuring RSA keys...")
	ensureRSAKey()
	greenlogger.ELogMessage("RSA keys confirmed to exist")

	greenlogger.LogMessage("Ensuring scouting schedule database...")
	ensureScoutDB(configs)
	greenlogger.ELogMessage("Schedule database confirmed to exist")

	greenlogger.LogMessage("Ensuring TBA API python package...")
	downloadAPIPackage()
	greenlogger.ELogMessage("API package downloaded")

	if !inTesting {
		greenlogger.LogMessage("Ensuring ip in configs...")
		configs.IP = recursivelyEnsureIP(configs.IP) //THIS DOES NOT CHECK FOR CONNECTIVITY BECAUSE PING IS STINKY IN GO
		greenlogger.ELogMessagef("IP %v confirmed ipv4", configs.IP)

		greenlogger.LogMessage("Ensuring domain name maps to IP...")
		configs.DomainName = recursivelyEnsureFunctionalDomain(&configs, configs.DomainName)
		greenlogger.ELogMessagef("Domain %v confirmed to match IP %v", configs.DomainName, configs.IP)
	} else {
		greenlogger.LogMessage("TEST MODE: Skipping ip and domain name ensuring...")
	}

	greenlogger.LogMessage("Ensuring python driver...")
	configs.PythonDriver = ensurePythonDriver(configs.PythonDriver)
	greenlogger.ELogMessagef("Python driver validated: %v", configs.PythonDriver)

	constants.CachedConfigs.PythonDriver = configs.PythonDriver

	greenlogger.LogMessage("Ensuring TBA API key...")
	configs.TBAKey = ensureTBAKey(configs)
	greenlogger.ELogMessagef("TBA key validated: %v", configs.TBAKey)

	constants.CachedConfigs.TBAKey = configs.TBAKey

	greenlogger.LogMessage("Ensuring Event key...")
	configs.EventKey, configs.EventKeyName = ensureEventKey(configs)
	greenlogger.ELogMessagef("Event key validated: %v", configs.EventKey)

	greenlogger.LogMessage("Writing all events to file...")
	lib.WriteEventsToFile()
	greenlogger.ELogMessage("All events written to file")

	if !constants.CustomEventKey {
		greenlogger.LogMessage("Writing event schedule to file...")
		lib.WriteScheduleToFile(configs.EventKey)
		greenlogger.ELogMessage("Event schedule written to file")

		lib.WriteTeamsToFile(configs.TBAKey, configs.EventKey)
		greenlogger.ELogMessagef("Teams at %v written to file", configs.EventKey)
	} else {
		configs.CustomEventConfigs = configCustomEvent(configs)
	}

	configs.SpreadSheetID = recursivelyEnsureSpreadsheetID(configs.SpreadSheetID)
	greenlogger.LogMessagef("Spreadsheet ID %v verified...", configs.SpreadSheetID)

	greenlogger.LogMessage("Ensuring slack settings...")
	configs.SlackConfigs = ensureSlackConfiguration(configs.SlackConfigs)
	greenlogger.ELogMessage("Slack configs verified")

	if !configs.LogConfigs.Configured {
		configs.LogConfigs.Configured = true
		configs.LogConfigs.Logging = true
		configs.LogConfigs.LoggingHttp = true
	} else if !configs.LogConfigs.Logging {
		greenlogger.ShutdownLogFile()
	}

	configFile, openErr := filemanager.OpenWithPermissions(configFilePath)
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem creating %v", configFilePath)
	}

	defer configFile.Close()

	encodeErr := yaml.NewEncoder(configFile).Encode(&configs)

	if encodeErr != nil {
		greenlogger.LogErrorf(encodeErr, "Problem encoding %v", configs)
	}

	constants.CachedConfigs = configs

	greenlogger.LogMessagef("Setup finished! If you need to alter configurations any further, please check %v", filepath.Join("setup", "greenscout.config.yaml"))
}

func retrieveGeneralConfigs() constants.GeneralConfigs {
	var genConfigs constants.GeneralConfigs

	configFile, openErr := os.Open(configFilePath)
	if openErr != nil && !errors.Is(openErr, os.ErrNotExist) {
		greenlogger.LogErrorf(openErr, "Problem opening %v", configFilePath)
	}
	defer configFile.Close()

	dataAsByte, readErr := io.ReadAll(configFile)

	if readErr != nil && configFile != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", configFile)
	}

	unmarshalErr := yaml.Unmarshal(dataAsByte, &genConfigs)
	if unmarshalErr != nil {
		greenlogger.LogErrorf(unmarshalErr, "Problem unmarshalling %v", dataAsByte)
	}
	return genConfigs
}

func ensurePythonDriver(existingDriver string) string {
	if validatePythonDriver(existingDriver) {
		return existingDriver
	}

	return recursivePythonValidation(true)
}

func recursivePythonValidation(firstRun bool) string {
	if firstRun {
		greenlogger.LogMessage("Enter the python driver installed on this machine (what you type to run a .py file from the command line): ")
	}

	var driver string
	_, scanErr := fmt.Scanln(&driver)

	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning python driver input")
	}

	if validatePythonDriver(driver) {
		return driver
	} else {
		greenlogger.LogMessagef("Sorry, %v doesn't appear to be a valid python driver. Please ensure you didn't make a typo!", driver)
		return recursivePythonValidation(false)
	}
}

func validatePythonDriver(driver string) bool {
	runnable := exec.Command(driver, "--version")

	out, execErr := runnable.Output()
	if execErr != nil && !strings.Contains(execErr.Error(), "no command") {
		greenlogger.LogErrorf(execErr, "Problem executing %v %v", driver, "--version")
	}

	return len(out) > 0 && strings.Contains(string(out), "Python")
}

func ensureSqliteDriver() string {
	if !validateSqliteDriver() {
		greenlogger.FatalLogMessage("Invalid sqlite3 driver! Please ensure it's in your path and accessable to this program. \n If you don't have sqlite, please download it at https://www.sqlite.org/")
	}

	return "sqlite3"
}

func validateSqliteDriver() bool {
	// Define the pattern to match 3.{someNumber}.{someNumber}
	pattern := `3\.\d+\.\d+`

	// This is so dumb why can't it just have sqlite in its name like every other -version arg
	re := regexp.MustCompile(pattern)

	runnable := exec.Command("sqlite3", "-version")

	out, execErr := runnable.Output()
	if execErr != nil {
		greenlogger.LogErrorf(execErr, "Problem executing %v %v", "sqlite3", "--version")
	}

	return re.FindString(string(out)) != ""
}

func validateTBAKey(configs constants.GeneralConfigs, key string) bool { //This is unreliable because TBA is very weird at times. It will sometimes just... let an incorrect api key authenticate.
	if key == "" {
		return false
	}

	runnable := exec.Command(configs.PythonDriver, "getStatus.py", key)

	out, execErr := runnable.Output()

	if execErr != nil {
		greenlogger.LogErrorf(execErr, "Problem executing %v %v %v", configs.PythonDriver, "getStatus.py", key)
	}

	return string(out) != "ERR"
}

func ensureTBAKey(configs constants.GeneralConfigs) string {
	if validateTBAKey(configs, configs.TBAKey) {
		return configs.TBAKey
	}

	return recursiveTBAKeyValidation(&configs, true)
}

func recursiveTBAKeyValidation(configs *constants.GeneralConfigs, firstRun bool) string {
	if firstRun {
		greenlogger.LogMessage("Enter your Blue Alliance API Key: ")
	}

	var key string
	_, scanErr := fmt.Scanln(&key)

	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning TBA key input")
	}

	if validateTBAKey(*configs, key) {
		return key
	} else {
		greenlogger.LogMessagef("Sorry, %v doesn't appear to be a valid TBA Key. ", key)
		return recursiveTBAKeyValidation(configs, false)
	}
}

func validateEventKey(configs constants.GeneralConfigs, key string) string {
	if string(key[0]) == "c" {
		constants.CustomEventKey = true
		return configs.EventKeyName
	}

	runnable := exec.Command(configs.PythonDriver, "getEvent.py", configs.TBAKey, key)

	out, execErr := runnable.Output()

	if execErr != nil {
		greenlogger.LogErrorf(execErr, "Problem executing %v %v %v %v", configs.PythonDriver, "getEvent.py", configs.TBAKey, key)
	}

	return string(out)
}

func ensureEventKey(configs constants.GeneralConfigs) (string, string) {
	response := validateEventKey(configs, configs.EventKey)
	if !strings.Contains(response, "ERR") {
		configs.EventKeyName = strings.ReplaceAll(strings.Trim(response, "\n"), "'", "")

		return configs.EventKey, configs.EventKeyName
	}

	return recursiveEventKeyValidation(&configs, true)
}

func recursiveEventKeyValidation(configs *constants.GeneralConfigs, firstRun bool) (string, string) {
	if firstRun {
		greenlogger.LogMessage("Please enter the Blue alliance Event Key to be used (ex: 2024mnst); For non-TBA events, please start your fake key with 'c' (ex: c2024gtch)")
	}

	var key string
	_, scanErr := fmt.Scanln(&key)

	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning TBA key input")
	}

	if name := validateEventKey(*configs, key); !strings.Contains(name, "ERR") {
		moveOldJson(key)
		return key, strings.ReplaceAll(strings.Trim(name, "\n"), "'", "")
	} else {
		greenlogger.LogMessagef("Sorry, %v doesn't appear to be a valid Event Key. ", key)
		return recursiveEventKeyValidation(configs, false)
	}
}

func SetEventKey(key string) bool {
	file, openErr := filemanager.OpenWithPermissions(configFilePath)
	if openErr != nil {
		greenlogger.LogErrorf(openErr, "Problem creating %v", configFilePath)
	}
	defer file.Close()

	var configs constants.GeneralConfigs

	decodeErr := yaml.NewDecoder(file).Decode(&configs)

	if decodeErr != nil {
		greenlogger.LogErrorf(decodeErr, "Problem decoding %v", configFilePath)
	}

	if name := validateEventKey(configs, key); !strings.Contains(name, "ERR") {
		configs.EventKey = key
		configs.EventKeyName = name

		encodeErr := yaml.NewEncoder(file).Encode(&configs)

		if encodeErr != nil {
			greenlogger.LogErrorf(decodeErr, "Problem encoding %v to %v", configs, configFilePath)
		}

		constants.CachedConfigs = configs

		return true
	}

	return false
}

func ensureInputtedJSON() {
	greenlogger.HandleMkdirAll(filepath.Join("InputtedJson", "In"))
	greenlogger.HandleMkdirAll(filepath.Join("InputtedJson", "Mangled"))
	greenlogger.HandleMkdirAll(filepath.Join("InputtedJson", "Written"))
	greenlogger.HandleMkdirAll(filepath.Join("InputtedJson", "Archive"))
	greenlogger.HandleMkdirAll(filepath.Join("InputtedJson", "Errored"))
	greenlogger.HandleMkdirAll(filepath.Join("InputtedJson", "Discarded"))
	greenlogger.HandleMkdirAll(filepath.Join("InputtedJson", "PitWritten"))
}

func moveOldJson(newKey string) {
	allJson, readErr := os.ReadDir(filepath.Join("InputtedJson", "Written"))

	if readErr != nil {
		greenlogger.LogErrorf(readErr, "Problem reading %v", filepath.Join("InputtedJson", "Written"))
	}

	for _, file := range allJson {
		if !strings.Contains(file.Name(), newKey) {
			newPath := filepath.Join("InputtedJson", "Archive", strings.Split(file.Name(), "_")[0])
			greenlogger.HandleMkdirAll(newPath)

			oldStr := filepath.Join("InputtedJson", "In", file.Name())
			oldLoc, openErr := os.Open(oldStr)
			if openErr != nil {
				greenlogger.LogErrorf(openErr, "Problem opening %v", oldStr)
			}

			newLoc, openErr := filemanager.OpenWithPermissions(filepath.Join(newPath, file.Name()))

			if openErr != nil {
				greenlogger.LogErrorf(openErr, "Problem creating %v", filepath.Join(newPath, file.Name()))
			}

			defer newLoc.Close()

			_, copyErr := io.Copy(newLoc, oldLoc)

			if copyErr != nil {
				greenlogger.LogErrorf(copyErr, "Problem copying %v to %v", oldStr, filepath.Join(newPath, file.Name()))
			}

			oldLoc.Close()

			removeErr := os.Remove(oldStr)

			if removeErr != nil {
				greenlogger.LogErrorf(removeErr, "Problem removing %v", oldStr)
			}
		}
	}
}

func ensureRSAKey() {
	if file, err := os.Open(filepath.Join("rsaUtil", "login-key.pub.pem")); errors.Is(err, os.ErrNotExist) {
		generateRSAPair()
		closeErr := file.Close()
		if closeErr != nil {
			greenlogger.LogErrorf(closeErr, "Problem closing %v", filepath.Join("rsaUtil", "login-key.pub.pem"))
		}
	} else if file, err := os.Open(filepath.Join("rsaUtil", "login-key.pem")); errors.Is(err, os.ErrNotExist) {
		generateRSAPair()
		closeErr := file.Close()
		if closeErr != nil {
			greenlogger.LogErrorf(closeErr, "Problem closing %v", filepath.Join("rsaUtil", "login-key.pem"))
		}
	}

	if rsaUtil.DecryptPassword(rsaUtil.EncodeWithPublicKey("test")) != "test" {
		greenlogger.FatalLogMessage("RSA keys mismatched! Look into this!")
	}

}

func generateRSAPair() {
	filename := filepath.Join("rsaUtil", "login-key")
	bitSize := 4096

	// Generate RSA key.
	key, keyGenErr := rsa.GenerateKey(rand.Reader, bitSize)
	if keyGenErr != nil {
		greenlogger.FatalLogMessage(keyGenErr.Error())
	}

	// Extract public component.
	pub := key.Public()

	// Encode private key to PKCS#1 ASN.1 PEM.
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	// Encode public key to PKCS#1 ASN.1 PEM.
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
		},
	)

	if err := filemanager.WriteFileWithPermissions(filename+".pem", keyPEM); err != nil {
		greenlogger.FatalLogMessage(err.Error())
	}

	// Write public key to file.
	if err := filemanager.WriteFileWithPermissions(filename+".pub.pem", pubPEM); err != nil {
		greenlogger.FatalLogMessage(err.Error())
	}
}

func ensureScoutDB(configs constants.GeneralConfigs) {

	_, err := os.Stat(filepath.Join("schedule", "scout.db"))
	if err != nil && os.IsNotExist(err) && filemanager.IsSudo() {
		greenlogger.FatalLogMessage("scout.db must still be created, please run go run main.go without sudo so you can alter its contents in the future.")
	}

	dbRef, openErr := sql.Open(configs.SqliteDriver, filepath.Join("schedule", "scout.db"))

	if openErr != nil {
		greenlogger.FatalLogMessage(openErr.Error())
	}

	var response any
	scanErr := dbRef.QueryRow("select count(1) from individuals").Scan(&response)
	if scanErr != nil {
		greenlogger.LogErrorf(scanErr, "Problem scanning SQL query result from %v", "select count(1) from individuals")
	}

	if response == nil {
		_, execErr := dbRef.Exec("CREATE TABLE individuals(uuid string not null primary key, username string, schedule string)")

		if execErr != nil {
			greenlogger.FatalLogMessage("Problem creating scouting schedule database")
		}
	}

	closeErr := dbRef.Close()
	if closeErr != nil {
		greenlogger.LogError(closeErr, "Problem closing scouting schedule database")
	}
}

func ensureSheetsAPI() {
	if _, err := os.Open("credentials.json"); errors.Is(err, os.ErrNotExist) {
		greenlogger.LogMessage("It appears there isn't a credentials.json file. Please follow the 'set up your environment' steps here: https://developers.google.com/sheets/api/quickstart/go#set_up_your_environment")
		greenlogger.LogMessage("Remember to publish your Google Cloud project before you create your tokens so that they don't expire after a few days!")
		os.Exit(1)
	}

	sheet.SetupSheetsAPI()
}

func recursivelyEnsureFunctionalDomain(configs *constants.GeneralConfigs, domain string) string {
	res, lookupErr := net.LookupIP(domain)

	if lookupErr != nil && domain != "" {
		greenlogger.FatalLogMessage("Unable to look up domain " + domain)
	}

	if len(res) > 0 && res[0].Equal(net.ParseIP(configs.IP)) {
		return domain
	}

	if domain == "" {
		greenlogger.LogMessagef("Please enter a domain name that redirects to the same IP address you have entered.")
	} else {
		greenlogger.LogMessagef("%v doesn't map to the configured IP address %v , Please enter a valid domain name:", domain, configs.IP)
	}

	var newAddr string
	_, scanErr := fmt.Scanln(&newAddr)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning domain name input")
	}

	return recursivelyEnsureFunctionalDomain(configs, newAddr)
}

func recursivelyEnsureIP(addr string) string {
	var ipFromAddr net.IP = net.ParseIP(addr)

	if ipFromAddr.To4() == nil {
		if addr == "" {
			greenlogger.LogMessage("Please enter the outward-facing IP address of this server.")
		} else {
			greenlogger.LogMessage("Error: " + addr + " isn't a valid IPv4 address. Please enter a valid one:")
		}

		var newAddr string
		_, scanErr := fmt.Scanln(&newAddr)

		if scanErr != nil {
			greenlogger.LogError(scanErr, "Problem scanning IP address input")
		}

		return recursivelyEnsureIP(newAddr)
	}

	return ipFromAddr.String()
}

func EnsureExternalConnectivity() {

	//Waits because sometimes there's a pane in order to give access to wifi on macs especially
	timer := time.NewTimer(10 * time.Second)
	<-timer.C

	greenlogger.LogMessage("Ensuring remote connectivity to server...")

	resp, httpErr := http.Get("https://" + constants.CachedConfigs.DomainName)

	if httpErr != nil {
		greenlogger.LogErrorf(httpErr, "Problem sending a GET to %v", "https://"+constants.CachedConfigs.DomainName)
	}

	if resp != nil {
		return
	}

	greenlogger.FatalLogMessage("Unable to externally connect to the server! Make sure all your ports are forwarded right and such things.")
}

func recursivelyEnsureSpreadsheetID(id string) string {
	if sheet.IsSheetValid(id) {
		sheet.SpreadsheetId = id
		return id
	}

	if id == "" {
		greenlogger.LogMessagef("Please enter a google sheets spreadsheet ID (the part in the url in between d/ and /edit ) that the account your token is associated with can edit.")
	} else {
		greenlogger.LogMessagef("Google Sheets spreadsheet ID %v is invalid, or you don't have permission to access it. Please enter an id of a spreadsheet that will work.", id)
	}
	var newId string
	_, scanErr := fmt.Scanln(&newId)
	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning spreadsheet ID input")
	}

	return recursivelyEnsureSpreadsheetID(newId)
}

func ensureDatabasesExist() { //this method only checks for the files, not their contents. Keeping those in line is up to the maintainer.
	_, authErr := os.Open(filepath.Join("GreenScout-Databases", "auth.db"))
	_, usersErr := os.Open(filepath.Join("GreenScout-Databases", "users.db"))

	if errors.Is(authErr, os.ErrNotExist) || errors.Is(usersErr, os.ErrNotExist) {
		greenlogger.LogMessage("One or both of your essential databases are missing. If you are a member of our organization on github, run")
		greenlogger.LogMessage(`git clone "https://github.com/TheGreenMachine/GreenScout-Databases.git" in this directory. If not, there are functions to generate your own directories in userDB.go and auth.go`)
		os.Exit(1)
	}
}

func downloadAPIPackage() { //always runs, just to be safe.
	runnable := exec.Command("pip", "install", "git+https://github.com/TBA-API/tba-api-client-python.git")
	_, execErr := runnable.Output()

	if execErr != nil && !strings.Contains(execErr.Error(), "exit status 1") {
		greenlogger.LogError(execErr, "Problem executing pip install git+https://github.com/TBA-API/tba-api-client-python.git")
		greenlogger.LogMessage("Attempting to run with pip3...")

		runnable = exec.Command("pip3", "install", "git+https://github.com/TBA-API/tba-api-client-python.git")
		_, err := runnable.Output()
		if err != nil && !strings.Contains(err.Error(), "exit status 1") {
			greenlogger.FatalError(err, "Could not install tba-api-client-python with pip or pip3! Please ensure you have pip in your $PATH")
		}
	}

}

func ensureSlackConfiguration(configs constants.SlackConfigs) constants.SlackConfigs {
	var configsToReturn constants.SlackConfigs = configs
	if !configs.Configured {
		greenlogger.LogMessage(`Enable slack integration? Type "yes" if so, anything else if not.`)
		var using string
		_, scanErr := fmt.Scanln(&using)

		if scanErr != nil {
			greenlogger.LogError(scanErr, "Problem scanning response to slack integration toggle")
		}

		configsToReturn.UsingSlack = strings.Contains(using, "yes")
	}

	if configsToReturn.UsingSlack {
		configsToReturn.BotToken = recursivelyEnsureSlackBotToken(configsToReturn.BotToken)
		configsToReturn.Channel = recursivelyEnsureSlackChannel(configsToReturn.Channel)
	}

	configsToReturn.Configured = true

	return configsToReturn
}

func recursivelyEnsureSlackBotToken(token string) string {
	if greenlogger.InitSlackAPI(token) {
		return token
	}

	if token == "" {
		greenlogger.LogMessage("Please enter a slack bot token. If you don't have one, follow the guide at slack/slack.md")
	} else {
		greenlogger.LogMessagef("Slack bot token %v is invalid, or it doesn't have the correct permissions. Please make sure you copied the bot token and followed the steps in slack/slack.md", token)
	}

	var inputtedToken string
	_, scanErr := fmt.Scanln(&inputtedToken)

	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning slack bot token input")
	}

	return recursivelyEnsureSlackBotToken(inputtedToken)
}

func recursivelyEnsureSlackChannel(channel string) string {
	if greenlogger.ValidateChannelAccess(channel) {
		return channel
	}

	if channel == "" {
		greenlogger.LogMessage("Please enter a slack channel name for the bot to write to.")
	} else {
		greenlogger.LogMessagef("Slack channel %v is invalid. Please make sure it is typed correctly, exists, and the bot has permission to write to it.", channel)
	}

	var inputtedChannel string
	_, scanErr := fmt.Scanln(&inputtedChannel)

	if scanErr != nil {
		greenlogger.LogError(scanErr, "Problem scanning slack channel input")
	}

	return recursivelyEnsureSlackChannel(inputtedChannel)
}

func configCustomEvent(configs constants.GeneralConfigs) constants.CustomEventConfigs {
	if !configs.CustomEventConfigs.Configured {
		greenlogger.LogMessage("Will your custom event have a schedule? Enter yes if so, anything else if not.")
		var response string
		_, scanErr := fmt.Scanln(&response)

		if scanErr != nil {
			greenlogger.LogError(scanErr, "Problem scanning custom event schedule confirmation")
		}

		configs.CustomEventConfigs.CustomSchedule = response == "yes"
	}

	if configs.CustomEventConfigs.CustomSchedule {
		greenlogger.LogMessage("Using schedule/schedule.json as the match schedule! Please make that it meets your non-TBA event schedule manually.")
	} else {
		schedule.WipeSchedule()
		greenlogger.LogMessage("Not using a schedule.")
	}

	configs.CustomEventConfigs.Configured = true

	return configs.CustomEventConfigs

}
