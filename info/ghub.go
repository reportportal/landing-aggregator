package info

import (
	"context"
	"fmt"
	"github.com/dghubble/sling"
	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2"
	"sort"
	"strings"
	"sync"
	"time"
	log "github.com/sirupsen/logrus"
	"github.com/reportportal/commons-go/commons"
)

const rpOrg string = "reportportal"

//GitHubStats is a structure for retrieving DockerHub tags
type GitHubStats struct {
	latestTags map[string]string
	tmu        *sync.RWMutex
}

//NewGitHubVersions creates new struct with default values
func NewGitHubVersions(ghToken string, includeBeta bool) *GitHubStats {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)

	ghClient := github.NewClient(oauth2.NewClient(context.Background(), ts))
	stats := &GitHubStats{
		tmu: &sync.RWMutex{},
	}

	//schedules updates of latest stats
	commons.Schedule(time.Hour, true, func() {
		repos, err := getRepos(ghClient)
		if nil != err {
			return
		}
		stats.tmu.Lock()
		defer stats.tmu.Unlock()
		stats.latestTags = getVersionsMap(ghClient, repos, includeBeta)
	})

	return stats
}

func getRepos(c *github.Client) ([]*github.Repository, error) {
	opt := &github.RepositoryListByOrgOptions{Type: "all"}
	repos, _, err := c.Repositories.ListByOrg(context.Background(), rpOrg, opt)
	return repos, err
}

func getVersionsMap(c *github.Client, repos []*github.Repository, includeBeta bool) (map[string]string) {
	versionMap := make(map[string]string, len(repos))

	for _, repo := range repos {
		var tagsRs []*github.RepositoryTag
		rq, _ := sling.New().Get(repo.GetTagsURL()).Request()

		_, err := c.Do(context.Background(), rq, &tagsRs)
		if nil != err {
			log.Error(err)
			continue
		}
		versions := version.Collection([]*version.Version{})
		for _, tag := range tagsRs {
			name := tag.GetName()

			//not a latest (we need explicit version), not a beta
			if "" != name && (includeBeta || !strings.Contains(strings.ToLower(name), "beta")) {
				v, err := version.NewVersion(name)
				if nil == err {
					versions = append(versions, v)
				}
			}
		}
		sort.Sort(versions)
		if len(versions) > 0 {
			versionMap[fmt.Sprintf("%s/%s", rpOrg, repo.GetName())] = versions[len(versions)-1].String()
		}
	}
	return versionMap
}

//GetLatestTags returns copy of latest versions/tags map
func (s *GitHubStats) GetLatestTags() map[string]string {
	s.tmu.RLock()
	defer s.tmu.RUnlock()

	tags := make(map[string]string, len(s.latestTags))
	for k, v := range s.latestTags {
		tags[k] = v
	}
	return tags
}
