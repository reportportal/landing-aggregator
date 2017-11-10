package info

import (
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
	"github.com/pkg/errors"
	"strings"
	"time"
	"sync/atomic"
	"gopkg.in/reportportal/commons-go.v1/commons"
	"google.golang.org/api/googleapi"
	log "github.com/sirupsen/logrus"
	"fmt"
)

//"https://www.youtube.com/watch?v=" + video.Id,

const (
	videosListSyncPeriod time.Duration = time.Second * 10
)

type YoutubeBuffer struct {
	youtube   *youtube.Service
	channelID string
	cacheSize int64

	info       atomic.Value
	searchETag string
	videosETag string
}

//TweetInfo represents short tweet version
type VideoInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	Duration    string `json:"duration,omitempty"`
	PublishedAt string `json:"published_at"`
}

func NewYoutubeVideosBuffer(channelID string, cacheSize int64, keyFile []byte) (*YoutubeBuffer, error) {
	jwtConfig, err := google.JWTConfigFromJSON(keyFile, youtube.YoutubeScope)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot build Youtube service")
	}

	client := jwtConfig.Client(context.TODO())
	srv, err := youtube.New(client)
	if nil != err {
		return nil, errors.Wrap(err, "Cannot build Youtube service")
	}
	buffer := &YoutubeBuffer{
		youtube:   srv,
		channelID: channelID,
		cacheSize: cacheSize,
	}

	//schedules updates of latest versions
	buffer.loadVideos()
	commons.Schedule(videosListSyncPeriod, true, func() {
		buffer.loadVideos()
	})
	return buffer, nil
}

func (y *YoutubeBuffer) GetVideos() []VideoInfo {
	return y.info.Load().([]VideoInfo)
}

func (y *YoutubeBuffer) loadVideos() {
	videos, err := y.getVideos()
	if nil != err {
		if googleapi.IsNotModified(err) {
			log.Info("No new videos find")
			return
		}
		log.Errorf("Error loading videos: %v", err)
	}
	log.Infof("Loaded %d video details", len(videos))
	y.info.Store(videos)
}
func (y *YoutubeBuffer) getVideos() ([]VideoInfo, error) {
	call := y.youtube.Search.List("snippet")
	call = call.ChannelId(y.channelID).Fields("items(id/videoId)").Type("video").MaxResults(y.cacheSize).IfNoneMatch(y.searchETag)
	searchRS, err := call.Do()
	if nil != err {
		return nil, err
	}
	y.searchETag = searchRS.Etag

	ids := make([]string, len(searchRS.Items))
	for i, item := range searchRS.Items {
		ids[i] = item.Id.VideoId
	}

	fmt.Println(y.videosETag)
	rs, err := y.youtube.Videos.List("snippet,contentDetails").Id(strings.Join(ids, ",")).IfNoneMatch(y.videosETag).MaxResults(y.cacheSize).Do()
	fmt.Println(rs.HTTPStatusCode)
	fmt.Println(rs.Header)
	if nil != err {
		return nil, err
	}
	y.videosETag = rs.Etag

	videos := make([]VideoInfo, len(rs.Items))
	for i, video := range rs.Items {
		videos[i] = VideoInfo{
			ID:          video.Id,
			Title:       video.Snippet.Title,
			PublishedAt: video.Snippet.PublishedAt,
			Duration:    video.ContentDetails.Duration,
			Thumbnail:   video.Snippet.Thumbnails.High.Url,
		}
	}

	return videos, nil

}
