package main

import (
	"fmt"
	"os"
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

// SearchWorker ...
type SearchWorker interface {
	findDefinition(word string, chErr chan WordSearchError) (*WordDefinition, *WordNotFound, error)
	instance() int
}

// Predicate ...
type Predicate func(w string) bool

// WordInfoFinder ...
type WordInfoFinder interface {
	Start(channels Channels)
}

type Channels struct {
	chWord chan string
	c      chan WordDefinition
	chNf   chan WordNotFound
	chDone chan bool
	chErr  chan WordSearchError
	chWrk  chan SearchWorker
}

// FindWordDefinitions ...
func FindWordDefinitions(chWord chan string, c chan WordDefinition, chNf chan WordNotFound, chDone chan bool, chErr chan WordSearchError, predicate Predicate) {
	chWorker := createWorkerChannel(25)

	channels := Channels{chWord: chWord, c: c, chNf: chNf, chDone: chDone, chErr: chErr, chWrk: chWorker}
	rapid := &RapidAPIWordDefFinder{}
	ff := WordInfoFinder(rapid)
	finders := []WordInfoFinder{ff}


	/// start a go routine for each finder and let them share the workers
	finderChannels := make([]chan string, 0)
	for _, finder := range finders {
		chw := make(chan string)
		finderChannels = append(finderChannels, chw)
		go func() {
			fchannels := channels
			fchannels.chWord = chw
			finder.Start(fchannels)
		}()
	}

	start(channels, finderChannels, predicate)

	return

}

func start(channels Channels, finderChannels []chan string, predicate Predicate) {

	for {
		select {
		case word := <-channels.chWord:
			if predicate(word) {
				//fmt.Printf("\nstart(): word: %s\n", word)
				for _, chFinder := range finderChannels {
					chFinder <- word
				}
			}

		case <-channels.chDone:
			fmt.Println("start(): got done. Exiting")
			return

		case searchErr := <-channels.chErr:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", searchErr.Error)
		}
	}

}
