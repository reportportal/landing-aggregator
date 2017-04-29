package info

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestToTweetInfo(t *testing.T) {
	now := time.Now().Round(time.Second)
	tweet := twitter.Tweet{Text: "hello world", User: &twitter.User{
		Name: "John"}, CreatedAt: now.Format(time.RubyDate)}

	tweetInfo := toTweetInfo(&tweet)

	assert.Equal(t, "John", tweetInfo.User, "User is incorrect")
	assert.Equal(t, now, tweetInfo.CreatedAt, "Date is incorrect")
	assert.Equal(t, "hello world", tweetInfo.Text, "Text is incorrect")

}
