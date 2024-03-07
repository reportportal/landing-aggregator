package info

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/reportportal/commons-go/v5/commons"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/youtube/v3"
)

//"https://www.youtube.com/watch?v=" + video.Id,

const (
	videosListSyncPeriod = time.Hour * 2
)

// YoutubeBuffer represents buffer of videos
type YoutubeBuffer struct {
	youtube   *youtube.Service
	channelID string
	cacheSize int64

	info       atomic.Value
	searchETag string
	videosETag string
}

// VideoInfo represents video details
type VideoInfo struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Thumbnail   string     `json:"thumbnail,omitempty"`
	Duration    string     `json:"duration,omitempty"`
	PublishedAt string     `json:"published_at"`
	Statistics  Statistics `json:"statistics,omitempty"`
}

// Statistics represents video statistics
type Statistics struct {
	CommentCount uint64 `json:"comment_count,omitempty"`
	LikeCount    uint64 `json:"like_count,omitempty"`
	ViewCount    uint64 `json:"view_count,omitempty"`
}

// NewYoutubeVideosBuffer creates new buffer of YouTube videos info
func NewYoutubeVideosBuffer(channelID string, cacheSize int, keyFile []byte) (*YoutubeBuffer, error) {
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
		cacheSize: int64(cacheSize),
	}

	//schedules updates of latest versions
	buffer.loadVideos()
	commons.Schedule(videosListSyncPeriod, true, func() {
		buffer.loadVideos()
	})
	return buffer, nil
}

// GetAllVideos returns all videos available in the buffer
func (y *YoutubeBuffer) GetAllVideos() []VideoInfo {
	return y.info.Load().([]VideoInfo)
}

// GetVideos returns slice with specified count of videos
func (y *YoutubeBuffer) GetVideos(c int) []VideoInfo {
	items, ok := y.info.Load().([]VideoInfo)
	if !ok {
		return []VideoInfo{}
	}
	if len(items) < c {
		return items
	}
	return items[0:c]
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
	call = call.
		ChannelId(y.channelID).
		Fields("items(id/videoId)").
		Order("date").
		Type("video").
		MaxResults(y.cacheSize).
		IfNoneMatch(y.searchETag)
	searchRS, err := call.Do()
	if nil != err {
		return nil, err
	}
	y.searchETag = searchRS.Etag

	ids := make([]string, len(searchRS.Items))
	for i, item := range searchRS.Items {
		ids[i] = item.Id.VideoId
	}
	rs, err := y.youtube.Videos.
		List("snippet,contentDetails,statistics").
		Id(strings.Join(ids, ",")).
		IfNoneMatch(y.videosETag).
		MaxResults(y.cacheSize).
		Do()
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
			Statistics: Statistics{
				CommentCount: video.Statistics.CommentCount,
				LikeCount:    video.Statistics.LikeCount,
				ViewCount:    video.Statistics.ViewCount,
			},
		}
	}

	return videos, nil

}
