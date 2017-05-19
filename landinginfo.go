package main

import (
	"github.com/caarlos0/env"
	"github.com/reportportal/commons-go/commons"
	"github.com/reportportal/landing-aggregator/info"
	"goji.io"
	"goji.io/pat"
	_ "net/http/pprof"

	"net/http"
	log "github.com/sirupsen/logrus"

	"os"
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

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{ForceColors: true})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)

}

func main() {
	go func() {
		log.Info("hello world")
		log.Info(http.ListenAndServe(":6060", nil))
	}()

	conf := loadConfig()
	twitsBuffer := info.BufferTweets(conf.ConsumerKey, conf.ConsumerSecret, conf.Token, conf.TokenSecret, conf.SearchTerm, conf.BufferSize)

	dockerHubTags := info.NewGitHubVersions(conf.GitHubToken, conf.IncludeBeta)

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/twitter"), func(w http.ResponseWriter, rq *http.Request) {
		if err := sendRS(http.StatusOK, info.GetTweets(twitsBuffer), w, rq); nil != err {
			log.Error(err)
		}
	})

	mux.HandleFunc(pat.Get("/versions"), func(w http.ResponseWriter, rq *http.Request) {
		if err := sendRS(http.StatusOK, dockerHubTags.GetLatestTags(), w, rq); nil != err {
			log.Error(err)
		}
	})

	buildInfo := &commons.BuildInfo{
		Version:   Version,
		Branch:    Branch,
		BuildDate: BuildDate,
	}
	mux.Handle(pat.Get("/info"), http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		commons.WriteJSON(http.StatusOK, buildInfo, w)
	}))

	//aggregate everything into on rs
	mux.Handle(pat.Get("/"), http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		rs := map[string]interface{}{}
		rs["latest_versions"] = dockerHubTags.GetLatestTags()
		rs["tweets"] = info.GetTweets(twitsBuffer)
		rs["build"] = buildInfo
		commons.WriteJSON(http.StatusOK, rs, w)
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
	log.Infof("Starting on port %d", conf.Port)
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
