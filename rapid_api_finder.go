package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type meaning struct {
	Noun, Verb, Adverb, Adjective string
}

type definition struct {
	Entry         string  `json:"entry"`
	Request       string  `json:"request"`
	Response      string  `json:"response"`
	Ipa           string  `json:"ipa"`
	Rersion       string  `json:"version"`
	Author        string  `json:"author"`
	Meaning       meaning `json:"meaning"`
	ResultCode    string  `json:"result_code"`
	ResultMessage string  `json:"result_msg"`
}

type ourWorker struct {
	Instance int
}

func (w ourWorker) findDefinition(word string, chErr chan WordSearchError) (*WordDefinition, *WordNotFound, error) {

	fmt.Printf("rapid_api_worker_%d searching for [%s]\n", w.instance(), word)
	url := "https://twinword-word-graph-dictionary.p.rapidapi.com/definition/?entry=" + word

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("x-rapidapi-host", "twinword-word-graph-dictionary.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", "5bbe0febcamshc64f3f714b85b22p1eedbdjsn732de6848c86")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not read response from server: %w", err)
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	//s := string(body)
	//fmt.Printf("RapidAPIWordDefFinder.FindDefinition: [%s]\n", s)
	var wdef definition
	err = json.Unmarshal(body, &wdef)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not read response from server: %w", err)
	}

	num, err := strconv.ParseInt(wdef.ResultCode, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not read response from server: %w", err)
	}

	if num != 200 {
		return nil, &WordNotFound{Word: word, Source: source, Message: wdef.ResultMessage}, nil
	}

	wordDefinition := make([]string, 0)

	if wdef.Meaning.Noun != "" {
		wordDefinition = append(wordDefinition, wdef.Meaning.Noun)
	}

	if wdef.Meaning.Verb != "" {
		wordDefinition = append(wordDefinition, wdef.Meaning.Verb)
	}

	if wdef.Meaning.Adverb != "" {
		wordDefinition = append(wordDefinition, wdef.Meaning.Adverb)
	}

	if wdef.Meaning.Adjective != "" {
		wordDefinition = append(wordDefinition, wdef.Meaning.Adjective)
	}

	wd := WordDefinition{Definitions: wordDefinition, Source: source, Word: word, Found: true}

	return &wd, nil, nil

}

func (w *ourWorker) instance() int {
	return w.Instance
}

// RapidAPIWordDefFinder ...
type RapidAPIWordDefFinder struct {
	Instance int
}

// Name ...
func (f *RapidAPIWordDefFinder) Name() string {
	return string("RapidAPIWordDefFinder")
}

var source string = "RapidAPI.com"

// Start ...
func (f *RapidAPIWordDefFinder) Start(channels Channels) {

	for {
		select {

		case <-channels.chDone:
			return

		case word := <-channels.chWord:
			//fmt.Printf("Finder %s searching for %s\n", f.Name(), word)
			/// Only run when we have enough workers
			worker := <-channels.chWrk
			//defer func() { channels.chWrk <- worker }()
			wd, wnf, err := worker.findDefinition(word, channels.chErr)
			channels.chWrk <- worker
			if err != nil {
				channels.chErr <- WordSearchError{
					Error:  err,
					Source: source,
					Word:   word,
				}
			}

			/// check if found
			if wnf != nil {
				channels.chNf <- *wnf
				continue
			}
			/// check if found
			if wd == nil {
				channels.chNf <- WordNotFound{Word: word, Source: source, Message: "Not found."}
				continue
			}
			//fmt.Printf("\nWrote def of [%s] to channel %v.\n", word, channels.c)
			channels.c <- *wd
		}
	}
}

func createWorkerChannel(size int) chan SearchWorker {
	chWrk := make(chan SearchWorker, size)
	for i := 0; i != size; i++ {

		w := SearchWorker(&ourWorker{Instance: i + 1})
		chWrk <- w
	}
	return chWrk
}
