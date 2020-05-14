package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

var subDir string = ".mydict"

// StoreDefinition ...
func StoreDefinition(w WordDefinition) error {

	defined, err := IsWordDefined(w.Word)
	if err != nil {
		return fmt.Errorf("Could not create directory [%s]: %w", wordSourcePath(w.Word, w.Source), err)
	}
	if defined {
		fmt.Printf("[%s] already defined.\n", w.Word)
		return nil
	}
	defDir := wordSourcePath(w.Word, w.Source)
	err = os.MkdirAll(defDir, 0755)
	if err != nil {
		return fmt.Errorf("Could not create directory [%s]: %w", defDir, err)
	}

	f, err := os.Create(filepath.Join(defDir, "definitions.txt"))
	if err != nil {
		return fmt.Errorf("Could not create file [%s]: %w", defDir, err)
	}

	defer f.Close()
	defer f.Sync()

	for _, wordDef := range w.Definitions {
		f.WriteString(wordDef)
		f.WriteString("\n\n")
	}
	return nil
}

func wordPath(word string) string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Could not get current user information: %v", err)
	}

	definitionPath := filepath.Join(usr.HomeDir, subDir)
	wordDir := filepath.Join(definitionPath, word)
	return wordDir
}

func wordSourcePath(word string, source string) string {
	return filepath.Join(wordPath(word), source)
}

func definitionFilename(word string, source string) string {
	wordDir := wordPath(word)
	defDir := filepath.Join(wordDir, source)
	defFile := filepath.Join(defDir, "definitions.txt")
	return defFile
}

// IsWordDefined ...
func IsWordDefined(word string) (bool, error) {
	path := wordPath(word)
	// fmt.Printf("[%s] has path [%s]\n", word, path)
	exists, err := directoryExists(path)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("Could not access directory [%s]: %w", path, err)
	}

	for _, f := range files {
		defined, err := IsDefinedBySource(word, f.Name())
		if err != nil {
			return false, err
		}
		return defined, nil
	}
	return false, nil
}

// IsDefinedBySource ...
func IsDefinedBySource(word string, source string) (bool, error) {
	defFile := definitionFilename(word, source)
	return fileExists(defFile)
}

func directoryExists(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("Could not determine existence of directory [%s]: %w", path, err)
	}
	return stat.IsDir(), nil
}

func fileExists(path string) (bool, error) {
	stat, err := os.Stat(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("Could not determine existence of file [%s]: %w", path, err)
	}
	return !stat.IsDir(), nil
}
