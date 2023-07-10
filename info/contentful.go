package info

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
)

// CmaClient is a client for Contentful Management API
type CmaClient struct {
	Token   string
	SpaceID string
	Limit   int
}

// NewsFeed is a struct for the Contentful News Feed
type NewsFeed struct {
	Items []struct {
		Fields struct {
			Text     string   `json:"text"`
			Hashtags []string `json:"hashtags"`
		} `json:"fields"`
	} `json:"items"`
}

// TwitterInfo is a struct for mapping the News Feed to Twitter
type TwitterInfo struct {
	Text     string   `json:"text"`
	Entities struct{} `json:"entities"`
}

var localCache = cache.New(2*time.Minute, 5*time.Minute)

// NewCma creates a new CmaClient
func NewCma(spaceID string, token string, limit int) *CmaClient {
	cma := &CmaClient{
		Token:   token,
		SpaceID: spaceID,
		Limit:   limit,
	}
	return cma
}

// FetchEntriesFromContentful fetches entries from Contentful
func FetchEntriesFromContentful(contentType string, spaceID string, token string, limit string) []byte {
	url := fmt.Sprintf("https://cdn.contentful.com/spaces/%s/entries?select=fields&content_type=%s&limit=%s", spaceID, contentType, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return nil
	}

	if token == "" {
		fmt.Printf("Error sending request: Environment variable CONTENTFUL_TOKEN not set.\n")
		return nil
	}

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return nil
	}

	// Debugging info
	// fmt.Printf("\n%s\n", body)

	return body
}

func mapEntriesToTwitterFeed(entry []byte) []*TwitterInfo {
	if entry == nil {
		fmt.Println("Error decoding JSON: response has empty body")
		return nil
	}

	var newsFeed NewsFeed
	err := json.Unmarshal(entry, &newsFeed)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	var tweets []*TwitterInfo

	// Map the fields to the Twitter structure
	for _, item := range newsFeed.Items {

		tweet := &TwitterInfo{
			Text:     item.Fields.Text,
			Entities: struct{}{},
		}

		tweets = append(tweets, tweet)
	}

	return tweets
}

// GetTwitterFeed provides a list of tweets from local cache or Contentful after mapping
func GetTwitterFeed(cma *CmaClient, count int) []*TwitterInfo {
	contentType := "newsFeed"
	cacheKey := contentType + cma.SpaceID

	if cachedEntry, found := localCache.Get(cacheKey); found {
		// Entry found in the cache, use it
		entry := cachedEntry.([]*TwitterInfo)

		// Debugging info
		// fmt.Printf("\nEntry fetched from local cache: \n%s\n", newsFeed)

		if count >= len(entry) {
			return entry
		}

		return entry[0:count]

	} else {
		// Entry not found in the cache, fetch it from Contentful
		body := FetchEntriesFromContentful(contentType, cma.SpaceID, cma.Token, strconv.Itoa(cma.Limit))

		// Map the fetched entry to a NewsFeed struct
		tweets := mapEntriesToTwitterFeed(body)

		// Store the fetched entry in the cache
		localCache.Set(cacheKey, tweets, cache.DefaultExpiration)

		// Debugging info
		// fmt.Printf("\nEntry fetched from Contentful: \n%s\n", newsFeed)

		if count >= len(tweets) {
			return tweets
		}

		return tweets[0:count]
	}
}
