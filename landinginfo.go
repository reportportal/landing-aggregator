package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi"
	"github.com/reportportal/commons-go/v5/commons"
	"github.com/reportportal/commons-go/v5/server"
	"github.com/reportportal/landing-aggregator/info"
	log "github.com/sirupsen/logrus"
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

	cma := info.NewCma(conf.CmaSpaceID, conf.CmaToken, conf.CmaLimit)

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
	router.Get("/info", http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		jsonRS(http.StatusOK, buildInfo, w)
	}))

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
	googleKeyFile, err := base64.StdEncoding.DecodeString(conf.GoogleAPIKeyFile)
	if nil != err {
		return nil, err
	}
	buf, err = info.NewYoutubeVideosBuffer(conf.YoutubeChannelID, conf.YoutubeBufferSize, googleKeyFile)
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

	GoogleAPIKeyFile string `env:"GOOGLE_API_KEY" envDefault:"ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsCiAgInByb2plY3RfaWQiOiAib3IyLW1zcS1lcG0tcnBwLWIyaXlsdSIsCiAgInByaXZhdGVfa2V5X2lkIjogImRjMzVjYjI1NmI3MWIwYTdmMDgyMWY0MWIyZDhmMTM3NzJlOWMzM2QiLAogICJwcml2YXRlX2tleSI6ICItLS0tLUJFR0lOIFBSSVZBVEUgS0VZLS0tLS1cbk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRREJud0RQUlZFUUVWelpcbnp1dHJCcnFlYkthMUhsMFJoWDRPTUM1VGNsTjZOblREZFh5OEhWWDUzQzJPUURvUkN2RndGdWMwcjlFa2dOMDNcbmVFb001SEJ2dVJRT3F0NUkwZWxoNG4yZGx2WUZ4OVZCZW4zSVBkU3FtSlpGelFNd0psYUkxa0tPTE1INm9aT0NcbjR4cGRVWUY3WWJxZzdoT1RySEhvK3ZWM2I4SXlSWHRWb1IvR3BxdnlnMGdqWTlLd3d4UlRTb3ZDWDNGUzVablhcbjRqakNwbnhuYTZHZkcxK1ZVWCtwRDdHOTB2NXVUNkRkTEhzWFZvZXpVaE5RdXlSdDJ5Y2FPODRORm5VLzAwZ1Bcbjhod1ByVE1NQnJaOFhFSHhvK1Fpdm1GOWRGS1drbHVPRlRQQStxRW14MzhwVUVrV2VZbXp4VzdndFNtQnNYbHJcbmpYcE1TaXdEQWdNQkFBRUNnZ0VBWHRsTjg3dUFwTzdrZmR4MERlZHJkeDFCb3pzZkcyeTZIaWd6SVhUQmVQNkJcblA1am54RjFZbDBCcFhxU083WGRmWStvTVZBNkcxU3Q5Y3VWdDNSZnhEb0hyVmU0Vld6WGRicktkbDV2eXBFMUtcbjVqc2pyL3ErR2Q0S3kySE5YSUtEWktBVlZZR09ld0U0K21iWExQeTNBZUtUb3E5T0RzcnN1RlZyOXhqYjJIUy9cbm1kR2NVRFF0WHQ5eWd5UjhmVklSTGNsUllmbGdjNy83c2NIbUlRNnA2M0ViYlhQRE1YRW44OTYvdWs4cDY5NzFcblFVK0wwR0FieEE2SFdSK3ZOQU84V24vR0dTRUZJL1ZqVVQ2V1g4ekdMcXg1ek1FdGNnL0xTYkM5UTRSd2FMQkdcbks4WlV4QXFHbklQYzlOR0ZOb0dGNmhHczFvNERLRmFqVVcyZmxuQlhOUUtCZ1FEc2hUanBpVWNud1NDS3VDcGNcbmEySHJIakVNUmFjOWs2U29uRG5ndDlkVTRjTkNDTGkyMHYycmU2WnQ2SmwzT25FN1RmTis1UFhMWDFWWHFBQmZcbmh3MXNXaU4wRGpiUHRGTUdDTW5lbURpK0lNKyt1RTloakhUUmJXKytwY2ZoM3FzNDVQQmozanNub2I2S1g1RmVcbmNmTXdQYjh3WnAzVDlTN25xc25kMi9PYmJ3S0JnUURSa1V1azVGSHJrU0p0UnAyem03YVE2ZmNKaEtHeU82Ym1cbjNHbHhObDVjNXlhdm1nY3BMWmNqU25Oa0lJSDdUNml3NjFIR2N1MTlxZFFjUjBKMW5HVWw2QlJjSEFhRmhJY05cbnM2ZVdXR1FqWFpSNWhTOE1IMTYwN0RFVXU0VVk3WlFBQUdNS1h5dlovd3g2R0dTam5rK3JMV3k4OHpkLzJ4eHBcbjZCV01JYTcrclFLQmdBTU5uVFIyanpLV0xhTmN5VDgwSzZsclZGckNNMng2RVhBVHhET0FiQWt1ZU9UTFZBY1lcbkppb21pSGwydlRScXpyZGpSRGRwSVRzazJlY3R4Z04xck5pdk9USHdWUWpOWFIwQTFBcEprTUh6am5yNXloeUtcblFaL0tkOXpRS3dwaFkzaHlqQi9kNkltVWJ1OCtXSlFOaUlRZzUrenFCak9NUUxUQTRhWTVocVdGQW9HQkFKdVJcbmFqL3JwY3hqSHRWVDJIbWVHL2FUVitsZTVkR3phb0J5R213S1doNUpFWFRGdUk4ZTR0VTF6VmNFc3JqbU4ybXVcbkpqUlUyR3V5aUZ5OW9WNUJUT3pJeldSYkFaUlgveEZ5emZOVGhuS2lZemVhWUlSMVBRNjlUdW4vRWh5aE1INlhcbnl1M0dISDFsVWRQSkM5eFNCdjRoYUZrVGk1MkVBQ0cyUVZpWElKcTVBb0dCQU5nTVk2UnpmdU9xbkZPdi9RQWVcbm55c2poYUZhSzJXMGx1aFdTRzdZcDJqc25PdjN2WUN1cmdYNEtHTVp6UmlpYnRSakZxV2ZuMFFWVHQzbWwvV2VcbjRsOTR3VzRwREl1V1EyZmdqOGFYd2hmYldQOGQxczZFZ1FNRUd4alZjWXBPSm5mcjRHb3hvZy9UUUY0b3VmaGdcbkh2djBVZUFZeVNsV0d0OW9TVHlzVGd4YlxuLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLVxuIiwKICAiY2xpZW50X2VtYWlsIjogInZpZXdlckBvcjItbXNxLWVwbS1ycHAtYjJpeWx1LmlhbS5nc2VydmljZWFjY291bnQuY29tIiwKICAiY2xpZW50X2lkIjogIjExNTc2ODM3NTQ5NzkwMzIzNTI3NyIsCiAgImF1dGhfdXJpIjogImh0dHBzOi8vYWNjb3VudHMuZ29vZ2xlLmNvbS9vL29hdXRoMi9hdXRoIiwKICAidG9rZW5fdXJpIjogImh0dHBzOi8vb2F1dGgyLmdvb2dsZWFwaXMuY29tL3Rva2VuIiwKICAiYXV0aF9wcm92aWRlcl94NTA5X2NlcnRfdXJsIjogImh0dHBzOi8vd3d3Lmdvb2dsZWFwaXMuY29tL29hdXRoMi92MS9jZXJ0cyIsCiAgImNsaWVudF94NTA5X2NlcnRfdXJsIjogImh0dHBzOi8vd3d3Lmdvb2dsZWFwaXMuY29tL3JvYm90L3YxL21ldGFkYXRhL3g1MDkvdmlld2VyJTQwb3IyLW1zcS1lcG0tcnBwLWIyaXlsdS5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbSIsCiAgInVuaXZlcnNlX2RvbWFpbiI6ICJnb29nbGVhcGlzLmNvbSIKfQo="`

	YoutubeBufferSize int    `env:"YOUTUBE_BUFFER_SIZE" envDefault:"10"`
	YoutubeChannelID  string `env:"YOUTUBE_CHANNEL_ID" envDefault:"UCsZxrHqLHPJcrkcgIGRG-cQ"`

	CmaToken   string `env:"CONTENTFUL_TOKEN"`
	CmaSpaceID string `env:"CONTENTFUL_SPACE_ID" envDefault:"1n1nntnzoxp4"`
	CmaLimit   int    `env:"CONTENTFUL_LIMIT" envDefault:"15"`
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
