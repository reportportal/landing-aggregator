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
	tweets, stream, err := searchTweets(searchTag, bufSize, client)
	if nil != err {
		log.Printf("Cannot load tweets: %s", err.Error())
	}
	for _, tweet := range tweets {
		buffer.Add(toTweetInfo(&tweet))
	}

	go func() {
		for message := range stream.Messages {
			tweet, ok := message.(*twitter.Tweet)
			//avoid retweets
			if ok && nil == tweet.RetweetedStatus {
				buffer.Add(toTweetInfo(tweet))
			}
		}
	}()
	return buffer
}

//toTweetInfo Build short tweet object
func toTweetInfo(tweet *twitter.Tweet) *TweetInfo {
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

func searchTweets(term string, count int, c *twitter.Client) ([]twitter.Tweet, *twitter.Stream, error) {

	//user timeline mode
	if strings.HasPrefix(term, "@") {
		u, _, err := c.Users.Show(&twitter.UserShowParams{
			ScreenName: strings.TrimPrefix(term, "@"),
		})
		if nil != err {
			log.Fatalf("Cannot load user: %s", err.Error())

		}

		searchTweetParams := &twitter.UserTimelineParams{
			UserID:          u.ID,
			Count:           count + 1,
			IncludeRetweets: twitter.Bool(false),
			ExcludeReplies:  twitter.Bool(true),
		}

		search, _, err := c.Timelines.UserTimeline(searchTweetParams)
		if nil != err {
			log.Fatalf("Cannot load user's tweets: %s", err.Error())
		}

		stream, err := c.Streams.User(&twitter.StreamUserParams{
			StallWarnings: twitter.Bool(true),
			With:          "user",
		})
		if nil != err {
			log.Fatalf("Cannot load user's stream: %s", err.Error())
		}

		return search, stream, err
	}

	// hashtag streaming mode
	streamFilterParams := &twitter.StreamFilterParams{
		StallWarnings: twitter.Bool(true),
		Track:         []string{term},
	}
	// search for existing tweets
	searchTweetParams := &twitter.SearchTweetParams{
		Query:           term,
		Count:           count,
		IncludeEntities: twitter.Bool(true),
	}

	// initially fill the buffer with existing tweets
	// useful for situation when there are rare updates
	search, _, err := c.Search.Tweets(searchTweetParams)
	if nil != err {
		log.Fatalf("Cannot load tweets: %s", err.Error())
	}

	stream, err := c.Streams.Filter(streamFilterParams)
	if nil != err {
		log.Fatalf("Cannot load tweets stream: %s", err.Error())
	}

	return search.Statuses, stream, err

}
