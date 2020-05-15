package main

import (
	"bufio"
	"fmt"
	"os"
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
		defined, err := IsWordDefined(w)
		if defined || err != nil {
			return false
		}
		return true
	}

	go FindWordDefinitions(chWords, chWordDefs, chNf, chDone, chErr, whichWords)

	go func() {
		count := 0
		nfCount := 0
		for {
			select {
			case wd, ok := <-chWordDefs:
				if !ok {
					fmt.Printf("\nStored %d Word Definitions and %d words not found. ", count, nfCount)
					chDone <- true
					return
				}
				fmt.Println("\nwordDef: ", wd)
				err := handleWordDef(&wd)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error handling [%s] from [%s]: \n\t%s", wd.Word, wd.Source, err)
				}
				count++

			case wnf := <-chNf:
				nfCount++
				fmt.Println("\nwordNotfound: ", wnf)

			case wErr := <-chErr:
				fmt.Printf("\nError: Word: [%s] Source: %s: \nMessage: %s\n", wErr.Word, wErr.Source, wErr.Error)

			case <-chDone:
				fmt.Printf("\nDONE: Stored %d Word Definitions and %d words not found. ", count, nfCount)
				close(chWordDefs)
				return

				// default:
				// 	fmt.Println("main gofunc: idle")
				// 	time.Sleep(100 * time.Millisecond)
			}

		}

	}()

	err := populateChannel(chWords, chDone, chErr)
	if err != nil {
		panic(fmt.Errorf("Could not populate channel with words: %w", err))
	}
}

func populateChannel(chWords chan string, chDone chan bool, chErr chan WordSearchError) error {

	// chBibleWords, err := createFileWordsChannel("bible.txt")
	chBibleWords, err := createFileWordsChannel("b.txt")
	if err != nil {
		return err
	}

	for {
		select {
		case word, ok := <-chBibleWords:
			word = strings.Trim(word, ".,:;?()!")
			if !ok {
				close(chWords)
				return nil
			}
			defined, err := IsWordDefined(word)
			if err != nil {
				return err
			}
			if !defined {
				chWords <- word
			}
		}

	}

}

func handleWordDef(wdef *WordDefinition) error {
	return StoreDefinition(*wdef)
}

func handleWordNotFound(path string, wnf *WordNotFound) error {
	return nil
}

func createFileWordsChannel(path string) (<-chan string, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// create the channels
	chWords := make(chan string)
	scanner := bufio.NewScanner(file)

	scanner.Split(bufio.ScanWords)

	go func() {
		defer file.Close()
		for scanner.Scan() {
			word := strings.TrimSpace(scanner.Text())
			if word != "" {
				chWords <- scanner.Text()
			}
		}
		close(chWords)
	}()
	return chWords, nil
}
