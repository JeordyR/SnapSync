package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/gregdel/pushover"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const version = "0.0.1"

var config snapsyncConfig
var log = logrus.New()

var daysOfWeek = map[string]time.Weekday{
	"Sunday":    time.Sunday,
	"Monday":    time.Monday,
	"Tuesday":   time.Tuesday,
	"Wednesday": time.Wednesday,
	"Thursday":  time.Thursday,
	"Friday":    time.Friday,
	"Saturday":  time.Saturday,
}

func main() {
	// Check for updates
	doSelfUpdate()

	// Parse input
	configFile := flag.String("config", "", "Path to configuration file.")
	flag.Parse()

	// Load configuration
	loadConfiguration(*configFile)

	// Setup logger
	file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0664)
	if err != nil {
		fmt.Println("Failed to open/creater log file ", config.LogFile)
		panic("Failed to open/create log file")
	}
	defer file.Close()

	log.SetOutput(file)

	// Send Pushover notification that sync starting
	sendPushoverMessage("Beginning snapsync...")
	log.Info("\n\n============= New Snapsync Execution =============")

	// Run touch if enabled
	if config.TouchEnabled == true {
		log.Info("Running touch...")
		_, err := runCommand("touch", false)
		if err != nil {
			log.Error("Failed to run touch with error: ", err)
			sendPushoverMessage("Failed to run touch, check log for errors.")
		} else {
			log.Info("Touch completed successfully.")
			sendPushoverMessage("Touch completed successfully.")
		}
	}

	// Get diff, panic if failed to get diff
	log.Info("Getting diff...")
	diffOutput, err := runCommand("diff", true)
	if err != nil {
		sendPushoverMessage("Failed to get diff results, check log for errors.")
		log.Panic("Failed to get diff results with error: ", err)
	}

	// Check if diff threashold is met and there are changes to sync
	log.Info("Parsing differences...")

	var added int = 0
	var removed int = 0

	for _, item := range strings.Split(diffOutput, "\n") {
		if strings.HasPrefix(item, "add") == true {
			added++
		} else if strings.HasPrefix(item, "remove") == true {
			removed++
		}
	}

	log.Info("Added files: ", added)
	log.Info("Removed files: ", removed)

	// Exit if delete threashold not met
	if removed <= config.DeleteThreashold {
		log.Info("Removed files ", removed, " meets threshold of ", config.DeleteThreashold, " continuing to sync.")
	} else {
		log.Info("Removed files ", removed, " does not meet threshold of ", config.DeleteThreashold, " not running sync.")
		sendPushoverMessage("Skipping sync, too many files removed: ", strconv.Itoa(removed), " threadshold is ", strconv.Itoa(config.DeleteThreashold))
		return
	}

	// Run sync
	log.Info("Running sync...")
	syncOutput, err := runCommand("sync", false)
	if err != nil {
		sendPushoverMessage("Sync failed, check logs for errors.")
		log.Panic("Sync failed with error: ", err)
	} else {
		// Confirm successful sync
		var completeLine string
		var complete bool = false
		var allOk bool = false

		// Loop through output looking for 100% completed
		for _, line := range strings.Split(syncOutput, "\n") {
			if allOk == true && complete == true {
				break
			}

			if strings.Contains(line, "100") == true && strings.Contains(line, "completed") == true {
				complete = true

				// Break up \r sub-lines and store the completed message
				for _, innerLine := range strings.Split(line, "\r") {
					if strings.Contains(innerLine, "completed") {
						completeLine = strings.TrimSpace(innerLine)
						break
					}
				}
			} else if strings.Contains(line, "Everything OK") == true {
				allOk = true
			}
		}

		// Log findings, panic if Sync failed
		if complete == true && allOk == true {
			log.Info("Sync completed successfully.")
			sendPushoverMessage("Sync ", completeLine)
		} else {
			sendPushoverMessage("Sync did not complete, output did not contain '100% completed' and 'Everything OK'")
			log.Panic("Sync did not complete, output did not contain '100% completed' and 'Everything OK'")
		}
	}

	// Run scrub if enabled
	if config.ScrubEnabled == true {
		log.Info("Running scrub...")
		scrubOutput, err := runCommand("scrub", false, "--percentage", config.ScrubPercentage, "--older-than", config.ScrubOlderThan)
		if err != nil {
			sendPushoverMessage("Scrub failed, check logs for errors.")
			log.Panic("Scrub failed with error: ", err)
		} else {
			var completeLine string
			var complete bool = false
			var allOk bool = false

			// Loop through output looking for 100% completed
			for _, line := range strings.Split(scrubOutput, "\n") {
				if allOk == true && complete == true {
					break
				}

				if strings.Contains(line, "100") == true && strings.Contains(line, "completed") == true {
					complete = true

					// Break up \r sub-lines and store the completed message
					for _, innerLine := range strings.Split(line, "\r") {
						if strings.Contains(innerLine, "completed") {
							completeLine = strings.TrimSpace(innerLine)
							break
						}
					}
				} else if strings.Contains(line, "Everything OK") == true {
					allOk = true
				}
			}

			// Log findings, panic if scrub failed
			if complete == true && allOk == true {
				log.Info("Scrub completed successfully.")
				sendPushoverMessage("Scrub ", completeLine)
			} else {
				log.Error("Scrub did not complete successfully, may have found errors, output did not contain '100% completed' and 'Everything OK'")
				sendPushoverMessage("Scrub did not complete, may have found errors, see log for scrub output.")
				log.Info("================ Scrub Output ================")
				log.Info(scrubOutput)
				log.Panic("Scrub failed.")
			}
		}
	}

	// Get snapraid status information if enabled
	if config.OutputStatus {
		log.Info("Getting status...")
		statusOutput, err := runCommand("status", false)
		if err != nil {
			sendPushoverMessage("Failed to get status of the array, check logs for errors.")
			log.Panic("Status failed with error: ", err)
		} else {
			var scrubAgeLine string
			var scrubPercentLine string
			var dangerLines []string

			// Loop through output looking for relevant lines
			for _, line := range strings.Split(statusOutput, "\n") {
				if strings.Contains(line, "The oldest block was scrubbed") == true {
					// Break up \r sub-lines and store the completed message
					for _, innerLine := range strings.Split(line, "\r") {
						if strings.Contains(innerLine, "The oldest block was scrubbed") == true {
							scrubAgeLine = strings.TrimSpace(innerLine)
						}
					}
				} else if strings.Contains(line, "of the array is not scrubbed") == true {
					// Break up \r sub-lines and store the completed message
					for _, innerLine := range strings.Split(line, "\r") {
						if strings.Contains(innerLine, "of the array is not scrubbed") == true {
							scrubPercentLine = strings.TrimSpace(innerLine)
						}
					}
				} else if strings.Contains(line, "DANGER!") == true {
					// Break up \r sub-lines and store the completed message
					for _, innerLine := range strings.Split(line, "\r") {
						if strings.Contains(innerLine, "DANGER!") == true {
							dangerLines = append(dangerLines, strings.TrimSpace(innerLine))
						}
					}
				}
			}

			// Log and notify scrub stat lines
			log.Info("Scrub Stats: ", scrubAgeLine)
			sendPushoverMessage("Scrub Stats: ", scrubAgeLine)
			log.Info("Scrub Percentage: ", scrubPercentLine)
			sendPushoverMessage("Scrub Percentage: ", scrubPercentLine)

			// Log and notify error lines if present
			if len(dangerLines) > 0 {
				for _, item := range dangerLines {
					log.Info("Array Error: ", item)
					sendPushoverMessage("Array Error: ", item)
				}
			} else {
				log.Info("No errors in the array.")
				sendPushoverMessage("No errors in the array.")
			}
		}
	}

	// Send Pushover notification that snapsync is done
	log.Info("Snapsync completed successfully")
	sendPushoverMessage("Snapsync completed successfully")
}

func doSelfUpdate() {
	v := semver.MustParse(version)

	latest, err := selfupdate.UpdateSelf(v, "JeordyR/SnapSync")
	if err != nil {
		println("Binary update failed:", err)
		return
	}

	if latest.Version.Equals(v) {
		log.Println("Current binary is the latest version", latest.Version)
	} else {
		log.Println("Successfully updated to version", latest.Version)
		fmt.Println("Release note:\n", latest.ReleaseNotes)
	}
}

func loadConfiguration(configFile string) {
	fmt.Println("Loading config...")

	// Confirm config file exists, check local dir if not provided
	if configFile != "" {
		_, err := os.Stat(configFile)
		if os.IsNotExist(err) {
			fmt.Printf("Config file: %v does not exist", configFile)
			panic("Provided configuration file does not exist or has bad permissions.")
		}

	} else {
		fmt.Println("No config file specified, checking local directory for snapsync.yaml...")

		_, err := os.Stat("snapsync.yaml")
		if os.IsNotExist(err) {
			fmt.Println("No config file provided and snapsync.yaml not found in execution directory.")
			panic("No config file provided and snapsync.yaml not found in execution directory.")
		} else {
			configFile = "snapsync.yaml"
		}
	}

	// Load config file
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println("Failed to load config file: ", err)
		panic("Failed to load config file")
	}

	// Read config file into snapsyncConfig object
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Faled to Unmarshal config: ", err)
		panic("Faled to Unmarshal config")
	}

	// Validate Required Variables
	if config.Executable == "" {
		fmt.Println("Executable not configured.")
		panic("Executable not configured.")
	}
	if config.LogFile == "" {
		fmt.Println("LogFile not configured.")
		panic("LogFile not configured.")
	}
	if config.DeleteThreashold == 0 {
		fmt.Println("DeleteThreashold not configured.")
		panic("DeleteThreashold not configured.")
	}

	// Validate Pushover keys present if enabled
	if config.PushoverEnabled == true {
		if config.PushoverAppKey == "" {
			fmt.Println("Pushover enabled but AppKey not provided, disabling pushover")
			config.PushoverEnabled = false
		} else if config.PushoverUserKey == "" {
			fmt.Println("Pushover enabled but UserKey not provided, disabling pushover")
			config.PushoverEnabled = false
		}
	}

	// Validate Scrub settings
	if config.ScrubEnabled == true {
		weekday := time.Now().Weekday()

		// Confirm required scrub settings are set
		if config.ScrubPercentage == "" {
			fmt.Println("Scrub enabled but ScrubPercentage not configured, disabling scrub.")
			config.ScrubEnabled = false
		} else if config.ScrubOlderThan == "" {
			fmt.Println("Scrub enabled but ScrubOlderThan not configured, disabling scrub.")
			config.ScrubEnabled = false
		} else if len(config.ScrubDaysOfWeek) == 0 {
			fmt.Println("Scrub enabled but ScrubDaysOfWeek not configured, disabling scrub.")
			config.ScrubEnabled = false
		}

		// Confirm day of week
		var dayOfWeekMatch bool = false
		for _, v := range config.ScrubDaysOfWeek {
			if daysOfWeek[v] == weekday {
				dayOfWeekMatch = true
				break
			}
		}

		if dayOfWeekMatch == false {
			fmt.Println("Day of week not in configured list for scrub, disabling scrub.")
			config.ScrubEnabled = false
		}
	}

	fmt.Printf("config: %+v\n", config)
}

func sendPushoverMessage(message string, messageArgs ...string) {
	if config.PushoverEnabled == false {
		return
	}
	log.Info("Sending pushover notification...")

	// Setup Pushover client and objects
	app := pushover.New(config.PushoverAppKey)
	recipient := pushover.NewRecipient(config.PushoverUserKey)

	// Setup message
	for _, v := range messageArgs {
		message = message + v
	}

	log.Info("Message: ", message)
	msg := pushover.NewMessage(message)

	// Send the message
	response, err := app.SendMessage(msg, recipient)
	if err != nil {
		log.Panic("Failed to send pushover message with error: ", err)
	}

	log.Debug("Pushover response: ", response)
}
