package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/denisbrodbeck/striphtmltags"
)

const (
	apiURL        = "https://a.4cdn.org/g/catalog.json"
	threadBaseURL = "http://boards.4chan.org/g/thread/"
)

type apithreadWrapperObject struct {
	Page    int           `json:"page"`
	Threads []pagedThread `json:"threads"`
}

type pagedThread struct {
	No  int    `json:"no"`
	Com string `json:"com"`
}

type aggregatedThread struct {
	threadName string //only really gets used for /*/ threads
	imageUrls  []string
	comments   []comment
}

type threadPage struct {
	Comments []comment `json:"posts"`
}

type comment struct {
	No  int    `json:"no"`
	Com string `json:"com"`
}

func main() {
	var apiWrapper []apithreadWrapperObject
	var wg sync.WaitGroup
	var threadChannel chan aggregatedThread
	var aggregatedThreads []aggregatedThread

	if threadChannel != nil {
		threadChannel = make(chan aggregatedThread)
	}

	response, err := http.Get(apiURL)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	err = json.NewDecoder(response.Body).Decode(&apiWrapper)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(apiWrapper); i++ {
		for j := 0; j < len(apiWrapper[i].Threads); j++ {
			wg.Add(1)
			go scrapeThread(apiWrapper[i].Threads[j], threadChannel)
		}
	}

	//wait for all the responses
	wg.Wait()

	for aggregatedThread := range threadChannel {
		aggregatedThreads = append(aggregatedThreads, aggregatedThread)
	}
	close(threadChannel)
	// threads := generateThreadUrls(response.Body)
	spew.Dump(aggregatedThreads)

}

func scrapeThread(thread pagedThread, returnChannel chan aggregatedThread) {
	var newAggregatedThread aggregatedThread
	var scrapedThreadComments threadPage
	var url strings.Builder
	url.WriteString(threadBaseURL)
	url.WriteString(strconv.Itoa(thread.No))
	url.WriteString(".json")

	response, err := http.Get(url.String())
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	err = json.NewDecoder(response.Body).Decode(&scrapedThreadComments)
	if err != nil {
		panic(err)
	}

	//first post is always contains the title
	newAggregatedThread.threadName = extractTitle(striphtmltags.StripTags(scrapedThreadComments.Comments[0].Com))
	newAggregatedThread.comments = scrapedThreadComments.Comments[0:]

	returnChannel <- newAggregatedThread
	return
}

func extractTitle(rawComment string) string {
	if len(rawComment) == 0 {
		return "empty title"
	}

	var title strings.Builder

	//official threads always have a / as their first character to denote their shorthand
	//some people reference /g/ so we ignore those
	if string(rawComment[0]) == "/" && string(rawComment[1]) != "g" {
		//we dont know the length of the abbreviation, so we need to loop untill we find the closing /
		//kinda like we do in lexical analysis on HTML parsers

		for i := 1; string(rawComment[i]) != "/"; i++ {
			title.WriteString(string(rawComment[i]))
		}

		return title.String()
	}

	//I dont care about garbage threads that are not global ones
	return "untitled thread"
}
