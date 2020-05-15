package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRapidAPIWordDefFinder_FindDefinition(t *testing.T) {
	type args struct {
	}
	ch := make(chan WordDefinition, 2)
	chWord := make(chan string, 2)
	chNf := make(chan WordNotFound, 2)
	chDone := make(chan bool, 1)
	chErr := make(chan WordSearchError, 2)
	chWrk := make(chan int)

	tests := []struct {
		name        string
		word        string
		channel     chan WordDefinition
		channelNf   chan WordNotFound
		channelErr  chan WordSearchError
		channelDone chan bool
		wantText    string
	}{
		{
			name:        "test 1",
			word:        "mask",
			channel:     ch,
			channelNf:   chNf,
			channelErr:  chErr,
			channelDone: chDone,

			wantText: "(nou) a covering to disguise or conceal the face",
		},
		// {
		// 	name:        "test 2",
		// 	word:        "adjective",
		// 	channel:     ch,
		// 	channelNf:   chNf,
		// 	channelErr:  chErr,
		// 	channelDone: chDone,
		// 	wantText:    "(nou) a word that expresses an attribute of something",
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &RapidAPIWordDefFinder{}

			go func() {
				fmt.Printf("Starting %s ...\n", tt.name)
				channels := Channels{chWord: chWord, c: tt.channel, chNf: tt.channelNf, chDone: tt.channelDone, chErr: tt.channelErr, chWrk: chWrk}
				StartFinder(f, channels)
				f.Start(channels)

			}()

			chWord <- tt.word
			tt.channelDone <- true
			wd := <-tt.channel
			fmt.Println("wd:", wd)
			assert.NotNil(t, wd)
			assert.Contains(t, wd.Definitions[0], tt.wantText)
			assert.Equal(t, true, wd.Found)

		})
	}
}

func TestRapidAPIWordDefFinder_FindDefinitionBadWord(t *testing.T) {

	f := WordInfoFinder(&RapidAPIWordDefFinder{})

	ch := make(chan WordDefinition, 10)
	chNf := make(chan WordNotFound, 10)
	chErr := make(chan WordSearchError, 10)
	chWord := make(chan string)
	chDone := make(chan bool)
	chWrk := make(chan int)

	go func() {
		channels := Channels{chWord: chWord, c: ch, chNf: chNf, chDone: chDone, chErr: chErr, chWrk: chWrk}
		fmt.Printf("Starting %s ...\n", "TestRapidAPIWordDefFinder_FindDefinitionBadWord")
		StartFinder(f, channels)

		fmt.Println("Started.")
		//assert.NotNil(t, err)
	}()

	go func() {
		for {
			select {

			case <-chDone:
				fmt.Printf("\nDONE! recv'd")
				return

			case er := <-chErr:
				fmt.Fprintf(os.Stderr, "Error finding [%s] using [%s]", er.Word, er.Error)

			case nf := <-chNf:
				fmt.Printf("\nCould not find [%s] at [%s]", nf.Word, nf.Source)
			case wd := <-ch:
				fmt.Println("sel wd:", wd)
				assert.NotNil(t, wd)
				assert.Equal(t, false, wd.Found)
				assert.Equal(t, []string{}, wd.Definitions)
			}
		}
	}()
	chWord <- "supercoderism"
	chDone <- true
	wnf := <-chNf
	fmt.Println("wnf:", wnf)
	assert.NotNil(t, wnf)
	assert.Equal(t, "Entry word not found", wnf.Message)

}
