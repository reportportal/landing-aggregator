package info

import (
	"testing"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestToTweetInfo(t *testing.T) {
	now := time.Now().String()
	tweet := twitter.Tweet{Text: "hello world", User: &twitter.User{
		Name: "John"}, CreatedAt: now}

	tweetInfo := toTweetInfo(&tweet)

	assert.Equal(t, "John", tweetInfo.User, "User is incorrect")
	assert.Equal(t, now, tweetInfo.CreatedAt, "Date is incorrect")
	assert.Equal(t, "hello world", tweetInfo.Text, "Text is incorrect")

}
