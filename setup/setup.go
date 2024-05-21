package setup

import (
	"GreenScoutBackend/constants"
	greenlogger "GreenScoutBackend/greenLogger"
	"GreenScoutBackend/lib"
	"GreenScoutBackend/rsaUtil"
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

func TotalSetup() {
	fmt.Println("Retreiving configs...")
	configs := retrieveGeneralConfigs()

	fmt.Println("Ensuring python driver...")
	configs.PythonDriver = ensurePythonDriver(configs.PythonDriver)

	fmt.Println("Ensuring sqlite3 driver...")
	configs.SqliteDriver = ensureSqliteDriver()

	fmt.Println("Ensuring TBA API key...")
	configs.TBAKey = ensureTBAKey(configs)

	fmt.Println("Ensuring Event key...")
	configs.EventKey, configs.EventKeyName = ensureEventKey(configs)

	fmt.Println("Ensuring InputtedJSON...")
	ensureInputtedJSON()

	constants.CachedConfigs = configs //yes i'm assigning this here and at the end don't question me

	fmt.Println("Writing event schedule to file...")
	lib.WriteScheduleToFile(configs.EventKey)

	fmt.Println("Ensuring RSA keys...")
	ensureRSAKey()

	fmt.Println("Ensuring scouting schedule database...")
	ensureScoutDB(configs)

	fmt.Println("Ensuring sheets API...")
	ensureSheetsAPI()

	lib.WriteTeamsToFile(configs.EventKey)

	fmt.Println("Ensuring ip in configs...")
	configs.IP = recursivelyEnsureIP(configs.IP)

	fmt.Println("Ensuring domain name maps to IP...")
	configs.DomainName = recursivelyEnsureFunctionalDomain(&configs, configs.DomainName)

	configs.SpreadSheetID = recursivelyEnsureSpreadsheetID(configs.SpreadSheetID)
	fmt.Println("Spreadsheet ID verified...")

	configs.PathToDatabases = "GreenScout-Databases" //this is the only one i'm not having the user enter mainly because git cloning is uniform
	ensureDatabasesExist()
	fmt.Println("Essential databases verified...")

	configFile, _ := os.Create(configFilePath)

	defer configFile.Close()

	err := yaml.NewEncoder(configFile).Encode(&configs)

	if err != nil {
		print(err.Error())
	}

	constants.CachedConfigs = configs

	greenlogger.LogMessage("Setup finished! If you need to alter configurations any further, please check setup/greenscout.config.yaml")

}

func retrieveGeneralConfigs() constants.GeneralConfigs {
	var genConfigs constants.GeneralConfigs

	configFile, _ := os.Open(configFilePath)
	defer configFile.Close()

	dataAsByte, readErr := io.ReadAll(configFile)

	if readErr != nil {
		fmt.Println(readErr)
	}

	yaml.Unmarshal(dataAsByte, &genConfigs)
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
		fmt.Println("Enter the python driver installed on this machine (what you type to run a .py file from the command line): ")
	}

	var driver string
	fmt.Scanln(&driver)

	if validatePythonDriver(driver) {
		return driver
	} else {
		fmt.Println("Sorry, " + driver + " doesn't appear to be a valid python driver. Please ensure you didn't make a typo!")
		return recursivePythonValidation(false)
	}
}

func validatePythonDriver(driver string) bool {
	runnable := exec.Command(driver, "--version")

	out, _ := runnable.Output()

	return len(out) > 0 && strings.Contains(string(out), "Python")
}

func ensureSqliteDriver() string {
	if !validateSqliteDriver() {
		panic("Invalid sqlite3 driver! Please ensure it's in your path and accessable to this program. \n If you don't have sqlite, please download it at https://www.sqlite.org/")
	}

	return "sqlite3"
}

func validateSqliteDriver() bool {
	// Define the pattern to match 3.{someNumber}.{someNumber}
	pattern := `3\.\d+\.\d+`

	// This is so dumb why can't it just have sqlite in its name like every other -version arg
	re := regexp.MustCompile(pattern)

	runnable := exec.Command("sqlite3", "-version")

	out, _ := runnable.Output()

	return re.FindString(string(out)) != ""
}

func validateTBAKey(configs constants.GeneralConfigs, key string) bool {
	if key == "" {
		return false
	}

	runnable := exec.Command(configs.PythonDriver, "getStatus.py", key)

	out, _ := runnable.Output()

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
		fmt.Println("Enter your Blue Alliance API Key: ")
	}

	var key string
	fmt.Scanln(&key)

	if validateTBAKey(*configs, key) {
		return key
	} else {
		fmt.Println("Sorry, " + key + " doesn't appear to be a valid TBA Key. ")
		return recursiveTBAKeyValidation(configs, false)
	}
}

func validateEventKey(configs constants.GeneralConfigs, key string) string {
	runnable := exec.Command(configs.PythonDriver, "getEvent.py", configs.TBAKey, key)

	out, _ := runnable.Output()

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
		fmt.Println("Please enter the Blue alliance Event Key to be used (ex: 2024mnst): ")
	}

	var key string
	fmt.Scanln(&key)

	if name := validateEventKey(*configs, key); !strings.Contains(name, "ERR") {
		moveOldJson(key)
		return key, strings.ReplaceAll(strings.Trim(name, "\n"), "'", "")
	} else {
		fmt.Println("Sorry, " + key + " doesn't appear to be a valid Event Key. ")
		return recursiveEventKeyValidation(configs, false)
	}
}

func SetEventKey(key string) bool {
	file, _ := os.Create(configFilePath)
	defer file.Close()

	var configs constants.GeneralConfigs

	yaml.NewDecoder(file).Decode(&configs)

	if name := validateEventKey(configs, key); !strings.Contains(name, "ERR") {
		configs.EventKey = key
		configs.EventKeyName = name

		yaml.NewEncoder(file).Encode(&configs)

		constants.CachedConfigs = configs

		// moveOldJson()

		return true
	}

	return false
}

func ensureInputtedJSON() {
	os.MkdirAll(filepath.Join("InputtedJson", "In"), os.ModePerm)
	os.MkdirAll(filepath.Join("InputtedJson", "Mangled"), os.ModePerm)
	os.MkdirAll(filepath.Join("InputtedJson", "Written"), os.ModePerm)
	os.MkdirAll(filepath.Join("InputtedJson", "Archive"), os.ModePerm)
	os.MkdirAll(filepath.Join("InputtedJson", "Errored"), os.ModePerm)
}

func moveOldJson(newKey string) {
	allJson, _ := os.ReadDir(filepath.Join("InputtedJson", "Written"))

	for _, file := range allJson {
		if !strings.Contains(file.Name(), newKey) {
			newPath := filepath.Join("InputtedJson", "Archive", strings.Split(file.Name(), "_")[0])
			os.MkdirAll(newPath, os.ModePerm)

			oldStr := filepath.Join("InputtedJson", "In", file.Name())
			oldLoc, _ := os.Open(oldStr)

			newLoc, _ := os.Create(filepath.Join(newPath, file.Name()))
			defer newLoc.Close()

			io.Copy(newLoc, oldLoc)

			oldLoc.Close()

			os.Remove(oldStr)
		}
	}
}

func ensureRSAKey() {
	if file, err := os.Open(filepath.Join("rsaUtil", "login-key.pub.pem")); errors.Is(err, os.ErrNotExist) {
		generateRSAPair()
		file.Close()
	} else if file, err := os.Open(filepath.Join("rsaUtil", "login-key.pem")); errors.Is(err, os.ErrNotExist) {
		generateRSAPair()
		file.Close()
	}

	if rsaUtil.DecryptPassword(rsaUtil.EncodeWithPublicKey("test")) != "test" {
		panic("RSA keys mismatched! Look into this!")
	}

}

func generateRSAPair() {
	filename := filepath.Join("rsaUtil", "login-key")
	bitSize := 4096

	// Generate RSA key.
	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		panic(err)
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

	if err := os.WriteFile(filename+".pem", keyPEM, 0700); err != nil {
		panic(err)
	}

	// Write public key to file.
	if err := os.WriteFile(filename+".pub.pem", pubPEM, 0755); err != nil {
		panic(err)
	}
}

func ensureScoutDB(configs constants.GeneralConfigs) {
	dbRef, _ := sql.Open(configs.SqliteDriver, filepath.Join("schedule", "scout.db"))

	var response any
	dbRef.QueryRow("select count(1) from individuals").Scan(&response)

	if response == nil {
		dbRef.Exec("CREATE TABLE individuals(uuid string not null primary key, username string, schedule string)")
	}
}

func ensureSheetsAPI() {
	if _, err := os.Open("credentials.json"); errors.Is(err, os.ErrNotExist) {
		fmt.Println("It appears there isn't a credentials.json file. Please follow the 'set up your environment' steps here: https://developers.google.com/sheets/api/quickstart/go#set_up_your_environment")
		fmt.Println("Remember to publish your Google Cloud project before you create your tokens so that they don't expire after a few days!")
		os.Exit(1)
	}

	sheet.SetupSheetsAPI()
}

func recursivelyEnsureFunctionalDomain(configs *constants.GeneralConfigs, domain string) string {
	res, _ := net.LookupIP(domain)

	if len(res) > 0 && res[0].Equal(net.ParseIP(configs.IP)) {
		return domain
	}

	fmt.Println("Error: " + domain + " doesn't map to the configured IP address " + configs.IP + ", Please enter a valid domain name:")

	var newAddr string
	fmt.Scanln(&newAddr)
	return recursivelyEnsureFunctionalDomain(configs, newAddr)
}

func recursivelyEnsureIP(addr string) string {
	var ipFromAddr net.IP = net.ParseIP(addr)

	if ipFromAddr.To4() == nil {

		fmt.Println("Error: " + addr + " isn't a valid IPv4 address. Please enter a valid one:")

		var newAddr string
		fmt.Scanln(&newAddr)
		return recursivelyEnsureIP(newAddr)
	}

	return ipFromAddr.String()
}

func EnsureExternalConnectivity() {

	//Waits because sometimes there's a pane in order to give access to wifi on macs especially
	timer := time.NewTimer(10 * time.Second)
	<-timer.C

	print("Ensuring remote connectivity to server...")

	resp, _ := http.Get("https://" + constants.CachedConfigs.DomainName)

	if resp != nil {
		return
	}

	fmt.Println("Unable to externally connect to the server! Make sure all your ports are forwarded right and such things.")
	os.Exit(1)
}

func recursivelyEnsureSpreadsheetID(id string) string {
	if sheet.IsSheetValid(id) {
		return id
	}

	fmt.Println("Google Sheets spreadsheet ID " + id + " is invalid, or you don't have permission to access it. Please enter an id of a spreadsheet that will work.")
	var newId string
	fmt.Scanln(&newId)

	return recursivelyEnsureSpreadsheetID(newId)
}

func ensureDatabasesExist() { //this method only checks for the files, not their contents. Keeping those in line is up to the maintainer.
	_, err := os.Open(filepath.Join("GreenScout-Databases", "auth.db"))
	_, err2 := os.Open(filepath.Join("GreenScout-Databases", "users.db"))

	if errors.Is(err, os.ErrNotExist) || errors.Is(err2, os.ErrNotExist) {
		println("One or both of your essential databases are missing. If you are a member of our organization on github, run")
		println(`git clone "https://github.com/TheGreenMachine/GreenScout-Databases.git" in this directory. If not, there are functions to generate your own directories in userDB.go and auth.go`)
		os.Exit(1)
	}
}
