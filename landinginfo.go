package main

import (
	"encoding/json"
	"github.com/caarlos0/env"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/gorilla/handlers"
	"github.com/reportportal/landing-aggregator/info"
	"goji.io"
	"goji.io/pat"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	conf := loadConfig()
	twitsBuffer := info.BufferTwits(conf.ConsumerKey, conf.ConsumerSecret, conf.Token, conf.TokenSecret, conf.HashTag, conf.BufferSize)

	dockerHubTags := info.NewDockerHubTags()

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/twitter"), func(w http.ResponseWriter, rq *http.Request) {

		tweets := []*twitter.Tweet{}
		twitsBuffer.Do(func(tweet interface{}) {
			tweets = append(tweets, tweet.(*twitter.Tweet))
		})
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(tweets)
		if nil != err {
			json.NewEncoder(w).Encode(map[string]string{"error": "cannot serialize response"})
		}
	})

	mux.HandleFunc(pat.Get("/versions"), func(w http.ResponseWriter, rq *http.Request) {

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		dockerHubTags.Do(func(tags map[string]string) {
			err := json.NewEncoder(w).Encode(tags)
			if nil != err {
				json.NewEncoder(w).Encode(map[string]string{"error": "cannot serialize response"})
			}
		})
	})

	mux.Use(func(next http.Handler) http.Handler {
		return handlers.LoggingHandler(os.Stdout, next)
	})

	// listen and server on mentioned port
	log.Printf("Starting on port %d", conf.Port)
	http.ListenAndServe(":"+strconv.Itoa(conf.Port), mux)

}

func loadConfig() *config {
	cfg := config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
	return &cfg
}

type config struct {
	Port int `env:"PORT" envDefault:"8080"`

	ConsumerKey    string `env:"CONSUMER,required"`
	ConsumerSecret string `env:"CONSUMER_SECRET,required"`
	Token          string `env:"TOKEN,required"`
	TokenSecret    string `env:"TOKEN_SECRET,required"`

	BufferSize int    `env:"BUFFER_SIZE" envDefault:"10"`
	HashTag    string `env:"HASHTAG" envDefault:"reportportal_io"`
}
