package info

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/reportportal/commons-go/commons"
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

	buffer, err := bufferTweets(searchTag, bufSize, client)
	if nil != err {
		log.Fatalf("Cannot load tweets: %s", err.Error())
	}

	return buffer
}

func bufferTweets(term string, count int, c *twitter.Client) (*buf.RingBuffer, error) {
	buffer := buf.New(count)

	//'follow' mode
	if strings.HasPrefix(term, "@") {

		//retrieve user to be followed
		u, _, err := c.Users.Show(&twitter.UserShowParams{
			ScreenName: strings.TrimPrefix(term, "@"),
		})
		if nil != err {
			return nil, err
		}

		//periodically load tweets
		go func() {
			searchTweetParams := &twitter.UserTimelineParams{
				UserID:          u.ID,
				Count:           count,
				IncludeRetweets: twitter.Bool(false),
				ExcludeReplies:  twitter.Bool(true),
			}

			//schedules updates of latest versions
			commons.Schedule(time.Minute*1, true, func() {

				//if buffer contains tweets already
				//ask twitter to return values starting from the last one
				last := buffer.Last()
				if nil != last {
					searchTweetParams.SinceID = last.(*TweetInfo).ID
				}
				loadTweets(c, searchTweetParams, buffer)
			})

		}()

	}

	//'hashtag/streaming' mode

	// search for existing tweets to initially fill the buffer
	rs, _, err := c.Search.Tweets(&twitter.SearchTweetParams{
		Query:           term,
		Count:           count,
		IncludeEntities: twitter.Bool(true),
	})
	if nil != err {
		return nil, err
	}

	// fill the buffer with initial set of tweets
	// useful for situation when there are rare updates
	for _, tweet := range rs.Statuses {
		buffer.Add(toTweetInfo(&tweet))
	}

	//setup streaming for updating buffer
	go func() {
		streamFilterParam := &twitter.StreamFilterParams{
			StallWarnings: twitter.Bool(true),
			Track:         []string{term},
		}

		for message := range streamTweets(c, streamFilterParam) {
			tweet, ok := message.(*twitter.Tweet)
			if ok {
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
		log.Fatalf("Streaming error: %s", err.Error())
	}
	return stream.Messages
}

//loadTweets loads found tweets into buffer
func loadTweets(c *twitter.Client, searchTweetParams *twitter.UserTimelineParams, buffer *buf.RingBuffer) {
	tweets, _, err := c.Timelines.UserTimeline(searchTweetParams)
	if nil != err {
		log.Printf("Cannot load tweets: %s", err.Error())
	}
	for _, tweet := range tweets {
		buffer.Add(toTweetInfo(&tweet))
	}
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
