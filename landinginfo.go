package main

import (
	"encoding/base64"
	"fmt"
	"github.com/caarlos0/env"
	"github.com/go-chi/chi"
	"github.com/reportportal/landing-aggregator/info"
	log "github.com/sirupsen/logrus"
	"gopkg.in/reportportal/commons-go.v1/commons"
	"gopkg.in/reportportal/commons-go.v1/server"
	"net/http"
	"os"
	"strconv"
)

const (
	defaultTwitterRSCount = 3
	defaultYoutubeRSCount = 3
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
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true, DisableTimestamp: false})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)

}

func main() {

	//load app config
	conf := loadConfig()

	//setup aggregators
	buildInfo := &commons.BuildInfo{
		Version:   Version,
		Branch:    Branch,
		BuildDate: BuildDate,
	}
	twitsBuffer := info.BufferTweets(conf.ConsumerKey, conf.ConsumerSecret, conf.Token, conf.TokenSecret, conf.SearchTerm, conf.BufferSize)
	ghAggr := info.NewGitHubAggregator(conf.GitHubToken, conf.IncludeBeta)

	youtubeBuffer, err := buildYoutubeBuffer(conf)
	if err != nil {
		log.Fatalf("Cannot init youtube buffer. %s", err)
	}

	router := chi.NewMux()

	//404 - NOT Found middleware
	router.NotFound(notFoundMiddleware)

	//CORS middleware, allow all domains
	router.Use(enableCORSMiddleware)

	//info endpoint
	router.Get("/info", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		server.WriteJSON(http.StatusOK, buildInfo, w)
	}))

	router.Get("/twitter", func(w http.ResponseWriter, rq *http.Request) {
		count := getQueryIntParam(rq, "count", defaultTwitterRSCount)
		if count > conf.BufferSize {
			jsonpRS(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("provided count exceed max allower value (%d)", conf.BufferSize)}, w, rq)
			return
		}
		jsonpRS(http.StatusOK, info.GetTweets(twitsBuffer, count), w, rq)

	})

	router.Get("/youtube", func(w http.ResponseWriter, rq *http.Request) {
		count := getQueryIntParam(rq, "count", defaultYoutubeRSCount)
		if count > conf.YoutubeBufferSize {
			jsonpRS(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("provided count exceed max allower value (%d)", conf.BufferSize)}, w, rq)
			return
		}
		jsonpRS(http.StatusOK, youtubeBuffer.GetVideos(count), w, rq)
	})

	router.Get("/versions", func(w http.ResponseWriter, rq *http.Request) {
		jsonpRS(http.StatusOK, ghAggr.GetLatestTags(), w, rq)
	})

	//GitHub-related routes
	router.Route("/github/*", func(ghRouter chi.Router) {
		ghRouter.Get("/stars", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
			server.WriteJSON(http.StatusOK, ghAggr.GetStars(), w)
		}))
		ghRouter.Get("/contribution", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
			server.WriteJSON(http.StatusOK, ghAggr.GetContributionStats(), w)
		}))
		ghRouter.Get("/issues", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
			server.WriteJSON(http.StatusOK, ghAggr.GetIssueStats(), w)
		}))
	})

	//aggregate everything into on rs
	router.Get("/", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		rs := map[string]interface{}{}
		rs["latest_versions"] = ghAggr.GetLatestTags()
		rs["tweets"] = info.GetTweets(twitsBuffer, defaultTwitterRSCount)
		rs["build"] = buildInfo

		ghStats := map[string]interface{}{
			"stars":              ghAggr.GetStars(),
			"contribution_stats": ghAggr.GetContributionStats(),
			"issue_stats":        ghAggr.GetIssueStats(),
		}
		rs["github"] = ghStats

		server.WriteJSON(http.StatusOK, rs, w)
	}))

	// listen and server on mentioned port
	log.Infof("Starting on port %d", conf.Port)
	http.ListenAndServe(":"+strconv.Itoa(conf.Port), router)

}

func jsonpRS(status int, body interface{}, w http.ResponseWriter, rq *http.Request) {
	jsonp := rq.URL.Query()["jsonp"]
	var err error
	if nil != jsonp && len(jsonp) >= 1 {
		//write JSONP
		err = server.WriteJSONP(status, body, jsonp[0], w)
	} else {
		//write JSON
		err = server.WriteJSON(status, body, w)
	}
	if nil != err {
		log.Error(err)
	}
}

func loadConfig() *config {
	cfg := config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
	return &cfg
}

func buildYoutubeBuffer(conf *config, ) (*info.YoutubeBuffer, error) {
	googleKeyFile, err := base64.StdEncoding.DecodeString(conf.GoogleAPIKeyFile)
	if nil != err {
		return nil, err
	}
	return info.NewYoutubeVideosBuffer(conf.YoutubeChannelID, conf.YoutubeBufferSize, googleKeyFile)
}

func getQueryIntParam(rq *http.Request, name string, def int) int {
	if pCount, err := strconv.Atoi(rq.URL.Query().Get(name)); nil == err {
		return pCount
	}
	return def
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

	GoogleAPIKeyFile  string `env:"GOOGLE_API_KEY" envDefault:"false"`
	YoutubeBufferSize int    `env:"YOUTUBE_BUFFER_SIZE" envDefault:"10"`
	YoutubeChannelID  string `env:"YOUTUBE_CHANNEL_ID,required"`
}

var notFoundMiddleware = func(w http.ResponseWriter, rq *http.Request) {
	server.WriteJSON(http.StatusNotFound, map[string]string{"error": "not found"}, w)
}

var enableCORSMiddleware = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, rq)
	})
}
