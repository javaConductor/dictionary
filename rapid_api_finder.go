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

// RapidAPIWordDefFinder ...
type RapidAPIWordDefFinder struct {
	Instance int
}

// FindDefinition ...
func (f *RapidAPIWordDefFinder) FindDefinition(word string, chErr chan WordSearchError, instance int) (*WordDefinition, *WordNotFound, error) {

	fmt.Printf("rapid_api_worker_%d searching for [%s]\n", instance, word)
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

	// s := string(body)
	// fmt.Printf("RapidAPIWordDefFinder.findDefinition: [%s]\n", s)
	var wdef definition
	err = json.Unmarshal(body, &wdef)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not read response from server: %w", err)
	}

	num, err := strconv.ParseInt(wdef.ResultCode, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not read result code from server: %w", err)
	}

	if num != 200 {
		return nil, &WordNotFound{Word: word, Source: f.Name(), Message: wdef.ResultMessage}, nil
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

	wd := WordDefinition{Definitions: wordDefinition, Source: f.Name(), Word: word, Found: true}

	return &wd, nil, nil

}

// Name ...
func (f *RapidAPIWordDefFinder) Name() string {
	return "rapidapi.com"
}

// String ...
func (f *RapidAPIWordDefFinder) String() string {
	return f.Name()
}
