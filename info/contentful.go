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

type NewsInfo struct {
	Text string `json:"text"`
}

var localCache = cache.New(2*time.Minute, 5*time.Minute)

func InitCma(spaceId string, token string, limit int) *CmaClient {
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

func GetNewsFeed(cma *CmaClient) []NewsInfo {
	contentType := "newsFeed"
	cacheKey := contentType + cma.SpaceId

	if cachedEntry, found := localCache.Get(cacheKey); found {
		// Entry found in the cache, use it
		newsFeed := cachedEntry.([]NewsInfo)

		// Debugging info
		// fmt.Printf("\nEntry fetched from local cache: \n%s\n", newsFeed)

		return newsFeed
	} else {
		// Entry not found in the cache, fetch it from Contentful
		body := FetchEntriesFromContentful(contentType, cma.SpaceId, cma.Token, strconv.Itoa(cma.Limit))

		// Map the fetched entry to a NewsFeed struct
		newsFeed := mapEntriesToNewsFeed(body)

		// Store the fetched entry in the cache
		localCache.Set(cacheKey, newsFeed, cache.DefaultExpiration)

		// Debugging info
		// fmt.Printf("\nEntry fetched from Contentful: \n%s\n", newsFeed)

		return newsFeed
	}
}

func mapEntriesToNewsFeed(entry []byte) []NewsInfo {
	if entry == nil {
		fmt.Println("Error decoding JSON: response has empty body")
		return nil
	}

	var jsonData map[string]interface{}
	decodingErr := json.Unmarshal(entry, &jsonData)
	if decodingErr != nil {
		fmt.Printf("Error decoding JSON: %v\n", decodingErr)
		return nil
	}

	items, ok := jsonData["items"].([]interface{})
	if !ok {
		fmt.Println("No items found")
		return nil
	}

	var newsFeed []NewsInfo
	for _, item := range items {
		itemData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		fieldsData, ok := itemData["fields"].(map[string]interface{})
		if !ok {
			continue
		}

		text, ok := fieldsData["text"].(string)
		if !ok {
			continue
		}

		newsFeed = append(newsFeed, NewsInfo{
			Text: text,
		})
	}

	return newsFeed
}
