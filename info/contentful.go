package info

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
)

type CmaClient struct {
	Token   string
	SpaceId string
}

type NewsInfo struct {
	Text string `json:"text"`
}

var contentfulCache = cache.New(5*time.Minute, 10*time.Minute)

func InitCma(spaceId string, token string) *CmaClient {
	cma := &CmaClient{
		Token:   token,
		SpaceId: spaceId,
	}
	return cma
}

func fetchEntryFromContentful(contentType string, spaceId string, token string) []byte {
	url := fmt.Sprintf("https://cdn.contentful.com/spaces/%s/entries?select=fields&content_type=%s", spaceId, contentType)
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

	// fmt.Printf("\n%s\n", body)

	return body
}

func GetEnteriesFromCache(contentType string, spaceId string, token string) []byte {

	cacheKey := contentType + spaceId

	if cachedEntry, found := contentfulCache.Get(cacheKey); found {
		// Entry found in the cache, use it
		entry := cachedEntry.([]byte)

		// fmt.Printf("Entry fetched from local cache: \n%s\n", entry)

		return entry
	} else {
		// Entry not found in the cache, fetch it from Contentful
		entry := fetchEntryFromContentful(contentType, spaceId, token)

		// Store the fetched entry in the cache
		contentfulCache.Set(cacheKey, entry, cache.DefaultExpiration)

		// fmt.Printf("Entry fetched from Contentful: \n%s\n", entry)

		return entry
	}

}

func GetNewsFeed(cma *CmaClient) []NewsInfo {
	contentType := "newsFeed"
	body := GetEnteriesFromCache(contentType, cma.SpaceId, cma.Token)

	var jsonData map[string]interface{}
	decodingErr := json.Unmarshal(body, &jsonData)
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
