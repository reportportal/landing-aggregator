package info

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type CmaClient struct {
	Token   string
	SpaceId string
}

type NewsInfo struct {
	Text string `json:"text"`
}

func InitCma(spaceId string, token string) *CmaClient {
	cma := &CmaClient{
		Token:   token,
		SpaceId: spaceId,
	}
	return cma
}

func GetNewsFeed(cma *CmaClient) []NewsInfo {
	url := fmt.Sprintf("https://cdn.contentful.com/spaces/%s/entries?select=fields&content_type=%s", cma.SpaceId, "newsFeed")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return nil
	}

	if cma.Token == "" {
		fmt.Printf("Can't make request. Environment variable CONTENTFUL_TOKEN not set.\n")
		return nil
	} else {
		req.Header.Add("Authorization", "Bearer "+cma.Token)
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

	var jsonData map[string]interface{}
	decodingErr := json.Unmarshal(body, &jsonData)
	if decodingErr != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
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
