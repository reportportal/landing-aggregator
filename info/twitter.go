package info

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/reportportal/landing-aggregator/buf"
	"log"
)

//TweetInfo represents short tweet version
type TweetInfo struct {
	Text      string `json:"text"`
	User      string `json:"user"`
	CreatedAt string `json:"created_at"`
}

//BufferTwits creates new synchronized auto-updating buffer of twits searched by provided hashtag
func BufferTwits(consumerKey string,
	consumerSecret string,
	tokenKey string,
	tokenSecret string, searchTag string, bufSize int) *buf.RingBuffer {

	// Configure and build twitter client
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(tokenKey, tokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	buffer := buf.New(bufSize)

	// search for existing tweets
	searchTweetParams := &twitter.SearchTweetParams{
		Query:           searchTag,
		Count:           bufSize,
		IncludeEntities: twitter.Bool(false),
	}

	// initially fill the buffer with existing tweets
	// useful for situation when there are rare updates
	search, _, err := client.Search.Tweets(searchTweetParams)
	if nil != err {
		log.Printf("Cannot load tweets: %s", err.Error())
	}
	for _, tweet := range search.Statuses {
		buffer.Add(toTweetInfo(&tweet))
	}

	go func() {
		params := &twitter.StreamFilterParams{
			Track:         []string{searchTag},
			StallWarnings: twitter.Bool(true),
		}
		stream, err := client.Streams.Filter(params)
		if nil != err {
			log.Printf("Cannot load tweets stream: %s", err.Error())
		}

		for message := range stream.Messages {
			tweet, ok := message.(*twitter.Tweet)
			if ok {
				buffer.Add(toTweetInfo(tweet))
			}
		}
	}()
	return buffer
}

//toTweetInfo Build short tweet object
func toTweetInfo(tweet *twitter.Tweet) *TweetInfo {
	return &TweetInfo{
		Text:      tweet.Text,
		CreatedAt: tweet.CreatedAt,
		User:      tweet.User.Name}
}
