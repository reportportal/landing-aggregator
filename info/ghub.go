package info

import (
	"context"
	"fmt"
	"github.com/dghubble/sling"
	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
	"github.com/reportportal/commons-go/commons"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"sort"
	"strings"
	"sync"
	"time"
)

const rpOrg string = "reportportal"

//GitHubStats is a structure for retrieving DockerHub tags
type GitHubStats struct {
	rmu   *sync.RWMutex
	repos []*github.Repository

	latestTags  map[string]string
	commitStats map[string][]*github.WeeklyCommitActivity
}

//Stars hold total count of stars and count of stars per repo
type Stars struct {
	Total int
	Repos map[string]int
}

//NewGitHubVersions creates new struct with default values
func NewGitHubVersions(ghToken string, includeBeta bool) *GitHubStats {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)

	ghClient := github.NewClient(oauth2.NewClient(context.Background(), ts))
	stats := &GitHubStats{
		rmu: &sync.RWMutex{},
	}

	//schedules updates of repos
	stats.loadRepos(ghClient)
	commons.Schedule(time.Hour*4, false, func() {
		stats.loadRepos(ghClient)
	})

	//schedules updates of latest stats
	commons.Schedule(time.Hour, true, func() {
		stats.latestTags = getVersionsMap(ghClient, stats.getRepos(), includeBeta)
	})

	//schedules updates of commit stats
	//commons.Schedule(time.Hour*24, true, func() {
	//	stats.commitStats = getCommitsStats(ghClient, stats.getRepos())
	//})

	return stats
}

//func getCommitsStats(c *github.Client, repos []*github.Repository) map[string][]*github.WeeklyCommitActivity {
//	repoCommits := make(map[string][]*github.WeeklyCommitActivity, len(repos))
//
//	for _, repo := range repos {
//		commons.Retry(5, time.Second*10, func() error {
//			stats, _, err := c.Repositories.ListCommitActivity(context.Background(), "reportportal", repo.GetName())
//			if nil != err {
//				log.Error(err)
//				return err
//			}
//
//			//log.Info("OKOKOK")
//			//for _, stat := range stats[len(stats)-12:] {
//			//	log.Println(stat)
//			//}
//			repoCommits[repo.GetName()] = stats
//			return nil
//		})
//
//	}
//
//	return repoCommits
//}

func getVersionsMap(c *github.Client, repos []*github.Repository, includeBeta bool) map[string]string {
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

	tags := make(map[string]string, len(s.latestTags))
	for k, v := range s.latestTags {
		tags[k] = v
	}
	return tags
}

//GetCommitStats returns copy of latest versions/tags map
func (s *GitHubStats) GetCommitStats() map[string][]*github.WeeklyCommitActivity {
	stats := make(map[string][]*github.WeeklyCommitActivity, len(s.latestTags))
	for k, v := range s.commitStats {
		stats[k] = v
	}
	return stats
}

//GetStars returns count of stars for each repository and total count
func (s *GitHubStats) GetStars() *Stars {
	repoStars := make(map[string]int, len(s.repos))
	total := 0
	for _, repo := range s.getRepos() {
		repoStars[repo.GetName()] = repo.GetStargazersCount()
		total += repo.GetStargazersCount()
	}
	return &Stars{Total: total, Repos: repoStars}
}

//getRepos returns copy of cached Github Repos
func (s *GitHubStats) getRepos() []*github.Repository {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	repos := make([]*github.Repository, len(s.repos))
	copy(repos, s.repos)
	return repos
}

//loadRepos loads repositories from GitHUB
func (s *GitHubStats) loadRepos(c *github.Client) {
	opt := &github.RepositoryListByOrgOptions{Type: "all"}
	repos, _, err := c.Repositories.ListByOrg(context.Background(), rpOrg, opt)
	if nil == err {
		s.rmu.Lock()
		defer s.rmu.Unlock()
		s.repos = repos

	}
}
