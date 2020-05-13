package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

func main() {

	startTime := time.Now()
	defer func() {
		stopTime := time.Now()
		fmt.Printf("\nTime taken:: %v\n", stopTime.Sub(startTime))
	}()
	chWords := make(chan string)
	chWordDefs := make(chan WordDefinition)
	chNf := make(chan WordNotFound)
	chDone := make(chan bool)
	chErr := make(chan WordSearchError)

	var whichWords Predicate
	whichWords = func(w string) bool {
		return true
	}

	//	b, e := os.OpenFile("/usr/share/dict/words", os.O_RDONLY, os.ModePerm)
	b, e := os.OpenFile("words", os.O_RDONLY, os.ModePerm)
	if e != nil {
		panic(e)
	}
	reader := io.Reader(b)

	go FindWordDefinitions(chWords, chWordDefs, chNf, chDone, chErr, whichWords)

	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Could not get user information: %w", err)
	}

	path := filepath.Join(usr.HomeDir, ".mydict")

	go func() {
		for {
			select {
			case wd := <-chWordDefs:
				fmt.Println("\nwordDef: ", wd)
				err := handleWordDef(path, &wd)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error handling [%v] from [%s]: %w", wd.Word, wd.Source, err)
				}

			case wnf := <-chNf:
				fmt.Println("\nwordNotfound: ", wnf)

			case <-chDone:
				return
			}
		}
	}()

	r := bufio.NewReader(reader)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if io.EOF == err {

				time.Sleep(100 * time.Millisecond)
				for {
					if len(chWords) == 0 {
						time.Sleep(100 * time.Millisecond)
						break
					}
				}
				chDone <- true
				break
			}
			// panic(err)
		}

		word := strings.TrimSpace(string(line))
		if word != "" {
			chWords <- word
		}
	}

}

func handleWordDef(path string, wdef *WordDefinition) error {

	// open the file
	// word dir
	wordDir := filepath.Join(path, wdef.Word)
	defDir := filepath.Join(wordDir, wdef.Source)

	exists, err := directoryExists(defDir)
	if err != nil {
		return fmt.Errorf("Could not read directory [%s]: %w", defDir, err)
	}
	if !exists {
		err := os.MkdirAll(defDir, 0755)
		if err != nil {
			return fmt.Errorf("Could not create directory [%s]: %w", defDir, err)
		}
	}
	defFile := filepath.Join(defDir, "definitions.txt")

	f, err := os.Create(defFile)
	if err != nil {
		return fmt.Errorf("Could not open create directory [%s]: %w", defDir, err)
	}

	if err != nil {
		return err
	}
	defer f.Close()
	defer f.Sync()

	for _, wordDef := range wdef.Definitions {
		f.WriteString(wordDef)
		f.WriteString("\n\n")
	}

	return nil
}

func handleWordNotFound(path string, wnf *WordNotFound) error {
	return nil
}

func directoryExists(path string) (bool, error) {

	stat, err := os.Stat(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return stat.IsDir(), nil
}
