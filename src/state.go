package main

import (
	"log"
	"os"
)

type State bool

// Init() writes forcast state default value for the state file. "0" means bad weather next 24 hours
func (s State) Init(stateFilePath string) {
	d := []byte("0")
	err := os.WriteFile(stateFilePath, d, 0644)
	if err != nil {
		log.Println("ERROR: can't write to Status file", err)
	}
	log.Println("INFO: Status file initialized with 0 value")
}

// Set() writes state to the state file
func (s State) Set(b bool, path string) {
	if b {
		d := []byte("1")
		err := os.WriteFile(path, d, 0644)
		if err != nil {
			log.Println("ERROR: can't write to Status file", err)
		}
	} else {
		d := []byte("0")
		err := os.WriteFile(path, d, 0644)
		if err != nil {
			log.Println("ERROR: can't write to Status file", err)
		}
	}
	log.Println("INFO: Status file updated with", b)
}

// isGood() returns true if state file contains "1" and false when "0"
func (s State) isGood(path string) bool {
	dat, err := os.ReadFile(path)
	if err != nil {
		log.Println("ERROR: can't read from Status file", err)
	}

	if string(dat) == "0" {
		return false
	} else {
		return true
	}
}
