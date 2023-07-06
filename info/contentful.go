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

type CmaClient struct {
	Token   string
	SpaceId string
	Limit   int
}

type NewsFeed struct {
	Items []struct {
		Fields struct {
			Text     string   `json:"text"`
			Hashtags []string `json:"hashtags"`
		} `json:"fields"`
	} `json:"items"`
}

type TwitterInfo struct {
	Text     string   `json:"text"`
	Entities struct{} `json:"entities"`
}

var localCache = cache.New(2*time.Minute, 5*time.Minute)

func NewCma(spaceId string, token string, limit int) *CmaClient {
	cma := &CmaClient{
		Token:   token,
		SpaceId: spaceId,
		Limit:   limit,
	}
	return cma
}

func FetchEntriesFromContentful(contentType string, spaceId string, token string, limit string) []byte {
	url := fmt.Sprintf("https://cdn.contentful.com/spaces/%s/entries?select=fields&content_type=%s&limit=%s", spaceId, contentType, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return nil
	}

	if token == "" {
		fmt.Printf("Error sending request: Environment variable CONTENTFUL_TOKEN not set.\n")
		return nil
	} else {
		req.Header.Add("Authorization", "Bearer "+token)
	}

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

func GetTwitterFeed(cma *CmaClient, count int) []*TwitterInfo {
	contentType := "newsFeed"
	cacheKey := contentType + cma.SpaceId

	if cachedEntry, found := localCache.Get(cacheKey); found {
		// Entry found in the cache, use it
		entry := cachedEntry.([]*TwitterInfo)

		// Debugging info
		// fmt.Printf("\nEntry fetched from local cache: \n%s\n", newsFeed)

		if count >= len(entry) {
			return entry
		} else {
			return entry[0:count]
		}

	} else {
		// Entry not found in the cache, fetch it from Contentful
		body := FetchEntriesFromContentful(contentType, cma.SpaceId, cma.Token, strconv.Itoa(cma.Limit))

		// Map the fetched entry to a NewsFeed struct
		tweets := mapEntriesToTwitterFeed(body)

		// Store the fetched entry in the cache
		localCache.Set(cacheKey, tweets, cache.DefaultExpiration)

		// Debugging info
		// fmt.Printf("\nEntry fetched from Contentful: \n%s\n", newsFeed)

		if count >= len(tweets) {
			return tweets
		} else {
			return tweets[0:count]
		}
	}
}
