package main

import (
	"fmt"
	"os"
	"time"
)

// WordDefinition  A message containing the definition of a word from a particular source
type WordDefinition struct {
	Word        string
	Definitions []string
	Source      string
	Found       bool
}

// WordNotFound  A message indicating a word could not be found at a containing the definition of a word from a particular source
type WordNotFound struct {
	Word    string
	Source  string
	Message string
}

// WordSearchError  A message indicating a word search error
type WordSearchError struct {
	Word   string
	Source string
	Error  error
}

// Predicate ...
type Predicate func(w string) bool

// WordInfoFinder ...
type WordInfoFinder interface {
	Name() string
	FindDefinition(word string, chErr chan WordSearchError, worker int) (*WordDefinition, *WordNotFound, error)
}

// Channels ...
type Channels struct {
	chWord chan string
	c      chan WordDefinition
	chNf   chan WordNotFound
	chDone chan bool
	chErr  chan WordSearchError
	chWrk  chan int
}

// FindWordDefinitions ...
func FindWordDefinitions(chWord chan string, c chan WordDefinition, chNf chan WordNotFound, chDone chan bool, chErr chan WordSearchError, predicate Predicate) {

	finders := []WordInfoFinder{&OxfordAPIFinder{}, &RapidAPIWordDefFinder{}}

	chWorker := createWorkerCapacityChannel(66)

	channels := Channels{chWord: chWord, c: c, chNf: chNf, chDone: chDone, chErr: chErr, chWrk: chWorker}

	/// start a go routine for each finder and let them share the workers
	wordChannels := make([]chan string, 0)
	doneChannels := make([]chan bool, 0)

	for _, finder := range finders {
		chWord := make(chan string)
		chDone := make(chan bool)
		wordChannels = append(wordChannels, chWord)
		doneChannels = append(doneChannels, chDone)
		fchannels := Channels{chWord: chWord, c: c, chNf: chNf, chDone: chDone, chErr: chErr, chWrk: chWorker}
		startFinder(finder, fchannels)
		time.Sleep(500 * time.Millisecond)
	}

	start(channels, wordChannels, doneChannels, predicate)
}

// dispatch words and Done messages to Finders' channels
func start(channels Channels, wordChannels []chan string, doneChannels []chan bool, predicate Predicate) {
	for {
		select {
		case word, ok := <-channels.chWord:
			if !ok {
				fmt.Printf("Word channel closed for definitionFinder")
				for _, chWord := range wordChannels {
					close(chWord)
				}
				return
			}
			// send word to each finder channel
			if predicate(word) {
				//fmt.Printf("\nstart(): word: %s\n", word)
				for _, chWord := range wordChannels {
					chWord <- word
				}
			}

		case <-channels.chDone:
			fmt.Println("definitionfinder.start(): got done. Exiting")
			//fmt.Printf("\nstart(): word: %s\n", word)
			for _, chDone := range doneChannels {
				chDone <- true
			}
			return

		case searchErr, ok := <-channels.chErr:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "\nError: %v\n", searchErr.Error)
		}
	}

}

// start a particular Finder
func startFinder(f WordInfoFinder, channels Channels) {
	fmt.Println("Starting finder:", f)

	go func() {
		for {
			select {

			case <-channels.chDone:
				fmt.Printf("Finder %s got DONE. Exiting.\n", f.Name())
				return

			case word, ok := <-channels.chWord:
				if !ok {
					fmt.Printf("Word channel closed for %s\n", f.Name())
					return
				}

				alreadyDefined, err := IsDefinedBySource(word, f.Name())
				if err != nil {
					channels.chErr <- WordSearchError{
						Error:  err,
						Source: f.Name(),
						Word:   word,
					}
					continue
				}
				if alreadyDefined {
					continue
				}
				/// Only run when worker available
				worker := <-channels.chWrk
				// fmt.Printf("Finder %s_%d searching for %s\n", f.Name(), worker, word)
				//defer func() { channels.chWrk <- worker }()
				go func() {
					wd, wnf, err := f.FindDefinition(word,
						channels.chErr,
						worker)
					channels.chWrk <- worker
					if err != nil {
						channels.chErr <- WordSearchError{
							Error:  err,
							Source: f.Name(),
							Word:   word,
						}
					}

					/// check if found
					if wnf != nil {
						channels.chNf <- *wnf
						return
					}
					/// check if found
					if wd == nil {
						channels.chNf <- WordNotFound{Word: word, Source: f.Name(), Message: "Definition missing."}
						return
					}
					//fmt.Printf("\nWrote def of [%s] to channel %v.\n", word, channels.c)
					channels.c <- *wd
				}()
			}
		}
	}()
}

func createWorkerCapacityChannel(capacity int) chan int {
	chWrk := make(chan int, capacity)
	for i := 0; i != capacity; i++ {
		chWrk <- i + 1
	}
	return chWrk
}
