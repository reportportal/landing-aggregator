package main

import (
	"github.com/caarlos0/env"
	"goji.io"
	"goji.io/pat"
	"log"
	"net/http"
	"strconv"
	"github.com/avarabyeu/goRP/commons"
	"github.com/reportportal/landing-aggregator/info"
)

var (
	// Branch contains the current Git revision. Use make to build to make
	// sure this gets set.
	Branch string

	// BuildDate contains the date of the current build.
	BuildDate string

	// Version contains version
	Version string
)

func main() {
	conf := loadConfig()
	twitsBuffer := info.BufferTwits(conf.ConsumerKey, conf.ConsumerSecret, conf.Token, conf.TokenSecret, conf.HashTag, conf.BufferSize)

	dockerHubTags := info.NewDockerHubTags()

	mux := goji.NewMux()

	mux.HandleFunc(pat.Get("/twitter"), func(w http.ResponseWriter, rq *http.Request) {

		tweets := []*info.TweetInfo{}
		twitsBuffer.Do(func(tweet interface{}) {
			tweets = append(tweets, tweet.(*info.TweetInfo))
		})
		commons.WriteJSON(http.StatusOK, tweets, w)
	})

	mux.HandleFunc(pat.Get("/versions"), func(w http.ResponseWriter, rq *http.Request) {
		dockerHubTags.Do(func(tags map[string]string) {
			commons.WriteJSON(http.StatusOK, tags, w)
		})
	})

	buildInfo := &commons.BuildInfo{
		Version:   Version,
		Branch:    Branch,
		BuildDate: BuildDate,
	}
	mux.Handle(pat.Get("/info"), http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		commons.WriteJSON(http.StatusOK, buildInfo, w)

	}))

	mux.Use(commons.NoHandlerFound(func(w http.ResponseWriter, rq *http.Request) {
		commons.WriteJSON(http.StatusNotFound, map[string]string{"error": "not found"}, w)
	}))

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
