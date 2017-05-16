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

	// initially fill the buffer with existing tweets
	// useful for situation when there are rare updates
	buffer, err := searchTweets(searchTag, bufSize, client)
	if nil != err {
		log.Fatalf("Cannot load tweets: %s", err.Error())
	}

	return buffer
}

func searchTweets(term string, count int, c *twitter.Client) (*buf.RingBuffer, error) {
	buffer := buf.New(count)
	var tweets []twitter.Tweet

	streamFilterParam := &twitter.StreamFilterParams{
		StallWarnings: twitter.Bool(true),
	}

	followMode := strings.HasPrefix(term, "@")
	if followMode {
		term = strings.TrimPrefix(term, "@")
		u, _, err := c.Users.Show(&twitter.UserShowParams{
			ScreenName: term,
		})
		if nil != err {
			return nil, err
		}

		streamFilterParam.Follow = []string{u.IDStr}

		rs, _, err := c.Timelines.UserTimeline(&twitter.UserTimelineParams{
			UserID:          u.ID,
			Count:           count + 1,
			IncludeRetweets: twitter.Bool(false),
			ExcludeReplies:  twitter.Bool(true),
		})
		if nil != err {
			return nil, err
		}
		tweets = rs

	} else {
		streamFilterParam.Track = []string{term}

		// search for existing tweets
		rs, _, err := c.Search.Tweets(&twitter.SearchTweetParams{
			Query:           term,
			Count:           count,
			IncludeEntities: twitter.Bool(true),
		})
		if nil != err {
			return nil, err
		}

		tweets = rs.Statuses
	}

	// initially fill the buffer with existing tweets
	// useful for situation when there are rare updates
	for _, tweet := range tweets {
		buffer.Add(toTweetInfo(&tweet))
	}

	//setup streaming for updating buffer
	go func() {
		for message := range streamTweets(c, streamFilterParam) {
			tweet, ok := message.(*twitter.Tweet)

			// do not allow retweets in follow user mode
			if ok && !(followMode && term != tweet.User.Name) {
				buffer.Add(toTweetInfo(tweet))
			}
		}
	}()
	return buffer, nil
}

// streamTweets starts streaming tweets from twitter
func streamTweets(c *twitter.Client, filter *twitter.StreamFilterParams) chan interface{} {
	stream, err := c.Streams.Filter(filter)
	if nil != err {
		panic(err)
	}
	return stream.Messages
}

//toTweetInfo Build short tweet object
func toTweetInfo(tweet *twitter.Tweet) *TweetInfo {
	t, err := time.Parse(time.RubyDate, tweet.CreatedAt)
	if err != nil {
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
