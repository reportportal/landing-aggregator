package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"

	"github.com/reportportal/commons-go/v5/commons"
	"github.com/reportportal/commons-go/v5/server"
	"github.com/reportportal/landing-aggregator/info"
	log "github.com/sirupsen/logrus"
)

const (
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

	cma := info.NewCma(conf.CmaSpaceID, conf.CmaToken, conf.CmaLimit)

	var mailchimpClient *info.MailchimpClient

	if conf.MailchimpApiKey == "false" {
		log.Error("Environment variable MAILCHIMP_API_KEY not set.")
	} else {
		mailchimpClient = info.NewMailchimpClient(conf.MailchimpApiKey)
		mailchimpClient.User = conf.MailchimpUser
		mailchimpClient.Timeout = time.Duration(conf.MailchimpTimeout) * time.Second
	}

	var ghAggr *info.GitHubAggregator
	if conf.GitHubToken == "false" {
		log.Error("Environment variable GITHUB_TOKEN not set.")
	} else {
		ghAggr = info.NewGitHubAggregator(conf.GitHubToken, conf.IncludeBeta)
	}

	var youtubeBuffer *info.YoutubeBuffer
	var err error
	if conf.YoutubeChannelID == "" {
		log.Error("Environment variable YOUTUBE_CHANNEL_ID not set")
	} else {
		youtubeBuffer, err = buildYoutubeBuffer(conf)
		if err != nil {
			log.Error("Cannot init youtube buffer. ", err)
		}
	}

	router := chi.NewMux()

	//404 - NOT Found middleware
	router.NotFound(notFoundMiddleware)

	//CORS middleware, allow all domains
	router.Use(enableCORSMiddleware)

	//info endpoint
	router.Get("/info", func(w http.ResponseWriter, rq *http.Request) {
		jsonRS(http.StatusOK, buildInfo, w)
	})

	router.Get("/twitter", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		count := getQueryIntParam(rq, "count", conf.CmaLimit)
		if count > conf.CmaLimit {
			jsonpRS(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("provided count exceed max allower value (%d)", conf.CmaLimit)}, w, rq)
			return
		}
		jsonpRS(http.StatusOK, info.GetTwitterFeed(cma, count), w, rq)
	}))

	router.Get("/youtube", func(w http.ResponseWriter, rq *http.Request) {
		count := getQueryIntParam(rq, "count", defaultYoutubeRSCount)
		if count > conf.YoutubeBufferSize {
			jsonpRS(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("provided count exceed max allower value (%d)", conf.YoutubeBufferSize)}, w, rq)
			return
		}
		jsonpRS(http.StatusOK, youtubeBuffer.GetVideos(count), w, rq)
	})

	router.Get("/versions", func(w http.ResponseWriter, rq *http.Request) {
		jsonpRS(http.StatusOK, ghAggr.GetLatestTags(), w, rq)
	})

	//GitHub-related routes
	router.Route("/github/", func(ghRouter chi.Router) {
		ghRouter.Get("/stars", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
			jsonRS(http.StatusOK, ghAggr.GetStars(), w)
		}))
		ghRouter.Get("/contribution", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
			jsonRS(http.StatusOK, ghAggr.GetContributionStats(), w)
		}))
		ghRouter.Get("/issues", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
			jsonRS(http.StatusOK, ghAggr.GetIssueStats(), w)
		}))
	})

	// aggregate everything into on rs
	router.Get("/", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		rs := map[string]interface{}{}

		rs["build"] = buildInfo
		rs["tweets"] = info.GetTwitterFeed(cma, conf.CmaLimit)
		rs["youtube"] = youtubeBuffer.GetVideos(defaultYoutubeRSCount)

		ghStats := map[string]interface{}{
			"stars":              ghAggr.GetStars(),
			"contribution_stats": ghAggr.GetContributionStats(),
			"issue_stats":        ghAggr.GetIssueStats(),
		}
		rs["github"] = ghStats
		rs["latest_versions"] = ghAggr.GetLatestTags()

		jsonRS(http.StatusOK, rs, w)
	}))

	// Mailchimp-related routes
	router.Route("/mailchimp/", func(mcRouter chi.Router) {
		mcRouter.Route("/lists/{listID}/members", func(mcListRouter chi.Router) {
			mcListRouter.Options("/", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
				w.Header().Add("Access-Control-Allow-Methods", "OPTIONS, POST")
				w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Add("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusOK)
			}))
			mcListRouter.Post("/", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
				member, err := mailchimpClient.AddSubscription(rq.Body, chi.URLParam(rq, "listID"))
				if err != nil {
					jsonRS(http.StatusBadRequest, map[string]string{"error": err.Error()}, w)
					return
				}
				jsonRS(http.StatusOK, member, w)
			}))
		})
	})

	// listen and server on mentioned port
	log.Infof("Starting on port %d", conf.Port)

	if err := http.ListenAndServe(":"+strconv.Itoa(conf.Port), router); nil != err {
		log.Fatal(err)
	}
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

func jsonRS(status int, body interface{}, w http.ResponseWriter) {
	if err := server.WriteJSON(status, body, w); nil != err {
		log.Error("Cannot respond", err)
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

func buildYoutubeBuffer(conf *config) (buf *info.YoutubeBuffer, err error) {
	defer func() {
		if r := recover(); r != nil {
			// find out exactly what the error was and set err
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
			// invalidate rep
			buf = nil
			// return the modified err and rep
		}
	}()

	if conf.GoogleApiKeyFile == "" {
		return nil, errors.New("environment variable GOOGLE_API_KEY not set")
	}
	buf, err = info.NewYoutubeVideosBuffer(conf.YoutubeChannelID, conf.YoutubeBufferSize, conf.GoogleApiKeyFile)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func getQueryIntParam(rq *http.Request, name string, def int) int {
	if pCount, err := strconv.Atoi(rq.URL.Query().Get(name)); nil == err {
		return pCount
	}
	return def
}

type config struct {
	Port int `env:"PORT" envDefault:"8080"`

	IncludeBeta bool   `env:"GITHUB_INCLUDE_BETA" envDefault:"false"`
	GitHubToken string `env:"GITHUB_TOKEN" envDefault:"false"`

	GoogleApiKeyFile string `env:"GOOGLE_API_KEY" envDefault:"false"`

	YoutubeBufferSize int    `env:"YOUTUBE_BUFFER_SIZE" envDefault:"10"`
	YoutubeChannelID  string `env:"YOUTUBE_CHANNEL_ID" envDefault:"false"`

	CmaToken   string `env:"CONTENTFUL_TOKEN"`
	CmaSpaceID string `env:"CONTENTFUL_SPACE_ID" envDefault:"1n1nntnzoxp4"`
	CmaLimit   int    `env:"CONTENTFUL_LIMIT" envDefault:"15"`

	MailchimpApiKey  string `env:"MAILCHIMP_API_KEY" envDefault:"false"`
	MailchimpUser    string `env:"MAILCHIMP_USER" default:"landing-aggregator"`
	MailchimpTimeout int    `env:"MAILCHIMP_TIMEOUT_SECONDS" default:"3"`
}

var notFoundMiddleware = func(w http.ResponseWriter, rq *http.Request) {
	jsonRS(http.StatusNotFound, map[string]string{"error": "not found"}, w)
}

var enableCORSMiddleware = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, rq)
	})
}
