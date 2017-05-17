package main

import (
	"github.com/caarlos0/env"
	"github.com/reportportal/commons-go/commons"
	"github.com/reportportal/landing-aggregator/info"
	"goji.io"
	"goji.io/pat"
	_ "net/http/pprof"

	"log"
	"net/http"
	"sort"
	"strconv"
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
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	conf := loadConfig()
	twitsBuffer := info.BufferTwits(conf.ConsumerKey, conf.ConsumerSecret, conf.Token, conf.TokenSecret, conf.SearchTerm, conf.BufferSize)

	dockerHubTags := info.NewGitHubVersions(conf.GitHubToken, conf.IncludeBeta)

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/twitter"), func(w http.ResponseWriter, rq *http.Request) {

		tweets := []*info.TweetInfo{}
		twitsBuffer.Do(func(tweet interface{}) {
			tweets = append(tweets, tweet.(*info.TweetInfo))
		})
		sort.Slice(tweets, func(i, j int) bool {
			return tweets[i].CreatedAt.After(tweets[j].CreatedAt)
		})
		if err := sendRS(http.StatusOK, tweets, w, rq); nil != err {
			log.Println(err.Error())
		}
	})

	mux.HandleFunc(pat.Get("/versions"), func(w http.ResponseWriter, rq *http.Request) {
		dockerHubTags.Do(func(tags map[string]string) {
			sendRS(http.StatusOK, tags, w, rq)
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

	//CORS, allow all domains
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			next.ServeHTTP(w, rq)
		})

	})

	// listen and server on mentioned port
	log.Printf("Starting on port %d", conf.Port)
	http.ListenAndServe(":"+strconv.Itoa(conf.Port), mux)

}

func sendRS(status int, body interface{}, w http.ResponseWriter, rq *http.Request) error {
	jsonp := rq.URL.Query()["jsonp"]
	if nil != jsonp && len(jsonp) >= 1 {
		return commons.WriteJSONP(status, body, jsonp[0], w)
	}
	return commons.WriteJSON(status, body, w)
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
	Port           int    `env:"PORT" envDefault:"8080"`
	ConsumerKey    string `env:"TWITTER_CONSUMER,required"`
	ConsumerSecret string `env:"TWITTER_CONSUMER_SECRET,required"`
	Token          string `env:"TWITTER_TOKEN,required"`
	TokenSecret    string `env:"TWITTER_TOKEN_SECRET,required"`
	BufferSize     int    `env:"TWITTER_BUFFER_SIZE" envDefault:"10"`
	SearchTerm     string `env:"TWITTER_SEARCH_TERM" envDefault:"@reportportal_io"`
	IncludeBeta    bool   `env:"GITHUB_INCLUDE_BETA" envDefault:"false"`
	GitHubToken    string `env:"GITHUB_TOKEN" envDefault:"false"`
}
