package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type metadata interface{}

type domain struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type sense struct {
	Definitions      []string `json:"definitions"`
	Domains          []domain `json:"domains"`
	Examples         []interface{}
	ID               string   `json:"id"`
	ShortDefinitions []string `json:"shortDefinitions"`
}

type entry struct {
	Etymologies     []string `json:"etymologies"`
	HomographNumber string   `json:"homographNumber"`
	Senses          []sense  `json:"senses"`
}

type lexicalCategory struct {
	ID   string
	Text string
}

type phrase struct {
	ID   string
	Text string
}

type pronunciation struct {
	AudioFile        string
	Dialects         []string
	PhoneticNotation string
	PhoneticSpelling string
}

type lexicalEntry struct {
	Entries         []entry
	Language        string
	LexicalCategory lexicalCategory
	Phrases         []phrase
	Pronunciations  []pronunciation
	Text            string
}

type result struct {
	ID             string `json:"id"`
	Language       string `json:"language"`
	Type           string `json:"type"`
	Word           string `json:"word"`
	LexicalEntries []lexicalEntry
}

type response struct {
	Error    string `json:"error"`
	ID       string `json:"id"`
	Metadata metadata
	Results  []result `json:"results"`
	Word     string   `json:"word"`
}

// OxfordAPIFinder ...
type OxfordAPIFinder struct {
	Instance int
}

// FindDefinition ...
func (f *OxfordAPIFinder) FindDefinition(word string, chErr chan WordSearchError, instance int) (*WordDefinition, *WordNotFound, error) {

	fmt.Printf("oxford_api_worker_%d searching for [%s]\n", instance, word)
	url := fmt.Sprintf("https://od-api.oxforddictionaries.com/api/v2/entries/en-gb/%s?strictMatch=false", word)
	//fmt.Printf("oxford_api_worker_%d [%s]\n", instance, url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("app_id", "e71f45fd")
	req.Header.Add("app_key", "4a0500a2616ab53726afd060238a7296")

	/// give it a minute
	time.Sleep(15 * time.Millisecond)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not connect to server: %w", err)
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	// s := string(body)
	// fmt.Printf("\nOxfordAPIFinder.FindDefinition: Got definition [%s]\n", s)
	var wdef response
	err = json.Unmarshal(body, &wdef)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not read response from server: %w", err)
	}
	if wdef.Error == "No entry found matching supplied source_lang, word and provided filters" {
		/// Create a WordNotFound and pass to not found channel
		return nil, &WordNotFound{Source: f.Name(), Word: word, Message: wdef.Error}, nil

	}

	fmt.Printf("Got [%v] for word [%s]", wdef, word)
	// array of definitions
	wordDefinition := make([]string, 0)

	// Loop thru results
	for _, result := range wdef.Results {

		if result.ID == word {
			for _, lentry := range result.LexicalEntries {
				for _, entry := range lentry.Entries {
					for _, sense := range entry.Senses {
						for _, definition := range sense.Definitions {
							trimmed := strings.TrimSpace(definition)
							if trimmed != "" {
								wordDefinition = append(wordDefinition, trimmed)
							}
						}
					}
				}
			}
		}
	}

	if len(wordDefinition) == 0 {
		/// Create a WordNotFound and pass to not found channel
		return nil, &WordNotFound{Source: f.Name(), Word: word, Message: "Definition missing."}, nil
	}
	wd := WordDefinition{Definitions: wordDefinition, Source: f.Name(), Word: word, Found: true}

	return &wd, nil, nil
}

// Name ...
func (f *OxfordAPIFinder) Name() string {
	return "oxforddictionaries.com"
}

// String ...
func (f *OxfordAPIFinder) String() string {
	return f.Name()
}
