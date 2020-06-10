package main

import (
	"os/exec"
)

// Runs snapsync with provided executable, command, and arguments, returns output of command
func runCommand(command string, ignoreErrors bool, args ...string) (string, error) {
	log.Debug("Snaprid Command: ", command)

	// Compile arguments into string slice
	arguments := []string{}

	arguments = append(arguments, command)

	for _, v := range args {
		arguments = append(arguments, v)
	}

	// Run external command and get output
	cmd := exec.Command(config.Executable, arguments...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		if ignoreErrors == true {
			output := string(out[:])
			return output, nil
		}

		log.Error("Encountered error in command ", command, ": ", err)
		return "", err
	}

	output := string(out[:])
	return output, nil
}
