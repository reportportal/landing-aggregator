package info

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/reportportal/landing-aggregator/buf"
	"log"
	"strings"
	"time"
)

//TweetInfo represents short tweet version
type TweetInfo struct {
	ID               int64                   `json:"id"`
	Text             string                  `json:"text"`
	User             string                  `json:"user"`
	CreatedAt        time.Time               `json:"created_at"`
	Entities         *twitter.Entities       `json:"entities,omitempty"`
	ExtendedEntities *twitter.ExtendedEntity `json:"extended_entities,omitempty"`
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

	// initially fill the buffer with existing tweets
	// useful for situation when there are rare updates
	tweets, streamFilter, err := searchTweets(searchTag, bufSize, client)
	if nil != err {
		log.Printf("Cannot load tweets: %s", err.Error())
	}
	for _, tweet := range tweets {
		buffer.Add(toTweetInfo(&tweet))
	}

	go func() {
		stream, err := client.Streams.Filter(streamFilter)
		if nil != err {
			log.Printf("Cannot load tweets stream: %s", err.Error())
		}

		for message := range stream.Messages {
			tweet, ok := message.(*twitter.Tweet)
			if ok {
				log.Printf("receive %s", tweet.Text)
				buffer.Add(toTweetInfo(tweet))
			}
		}
	}()
	return buffer
}

//toTweetInfo Build short tweet object
func toTweetInfo(tweet *twitter.Tweet) *TweetInfo {
	log.Println(tweet.CreatedAt)
	t, err := time.Parse(time.RubyDate, tweet.CreatedAt)
	if err != nil { // Always check errors even if they should not happen.
		panic(err)
	}

	return &TweetInfo{
		ID:               tweet.ID,
		Text:             tweet.Text,
		CreatedAt:        t,
		User:             tweet.User.Name,
		Entities:         tweet.Entities,
		ExtendedEntities: tweet.ExtendedEntities}
}

func searchTweets(term string, count int, c *twitter.Client) ([]twitter.Tweet, *twitter.StreamFilterParams, error) {
	params := &twitter.StreamFilterParams{
		StallWarnings: twitter.Bool(true),
	}

	if strings.HasPrefix(term, "@") {

		u, _, err := c.Users.Show(&twitter.UserShowParams{
			ScreenName: strings.TrimPrefix(term, "@"),
		})
		if nil != err {
			return nil, nil, err
		}

		params.Follow = []string{u.IDStr}
		searchTweetParams := &twitter.UserTimelineParams{
			UserID:          u.ID,
			Count:           count,
			IncludeRetweets: twitter.Bool(true),
		}

		search, _, err := c.Timelines.UserTimeline(searchTweetParams)
		return search, params, err
	}
	params.Track = []string{term}
	// search for existing tweets
	searchTweetParams := &twitter.SearchTweetParams{
		Query:           term,
		Count:           count,
		IncludeEntities: twitter.Bool(true),
	}

	// initially fill the buffer with existing tweets
	// useful for situation when there are rare updates
	search, _, err := c.Search.Tweets(searchTweetParams)
	if nil == err {
		return nil, params, err
	}
	return search.Statuses, params, err

}
