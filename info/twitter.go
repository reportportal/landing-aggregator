package info

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/reportportal/landing-aggregator/buf"
)

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
	search, _, _ := client.Search.Tweets(searchTweetParams)
	for _, tweet := range search.Statuses {
		buffer.Add(&tweet)
	}

	go func() {
		params := &twitter.StreamFilterParams{
			Track:         []string{searchTag},
			StallWarnings: twitter.Bool(true),
		}
		stream, _ := client.Streams.Filter(params)

		for message := range stream.Messages {
			tweet, ok := message.(*twitter.Tweet)
			if ok {
				buffer.Add(tweet)
			}
		}
	}()
	return buffer
}
