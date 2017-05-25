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

//StatRange represents statistics calculation range in weeks
type StatRange int

const (
	week       StatRange = 1
	month      StatRange = week * 4
	threeMonth StatRange = month * 3
)

var ranges = []StatRange{week, month, threeMonth}

const (
	rpOrg                      string        = "reportportal"
	repoSyncPeriod             time.Duration = time.Hour * 4
	versionsSyncPeriod         time.Duration = time.Hour
	contributorStatsSyncPeriod time.Duration = time.Hour * 12
	statsRetryPeriod           time.Duration = time.Second * 10
	statsRetryAttempts         int           = 5
)

//GitHubAggregator is a structure for retrieving DockerHub tags
type GitHubAggregator struct {
	c *github.Client

	rmu   *sync.RWMutex
	repos []*github.Repository

	ltmu       *sync.RWMutex
	latestTags map[string]string

	csmu              *sync.RWMutex
	contributionStats *ContributionStats
}

//ContributionStats contains aggregated info related to contribution to a organization repositories
type ContributionStats struct {
	Commits      map[StatRange]int `json:"commits"`
	Contributors map[StatRange]int `json:"unique_contributors"`
}

//Stars hold total count of stars and count of stars per repo
type Stars struct {
	Total int            `json:"total"`
	Repos map[string]int `json:"repos"`
}

//NewGitHubAggregator creates new struct with default values
func NewGitHubAggregator(ghToken string, includeBeta bool) *GitHubAggregator {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)

	ghClient := github.NewClient(oauth2.NewClient(context.Background(), ts))
	stats := &GitHubAggregator{
		c:    ghClient,
		rmu:  &sync.RWMutex{},
		ltmu: &sync.RWMutex{},
		csmu: &sync.RWMutex{},
	}

	//schedules updates of repos
	stats.loadRepos()
	commons.Schedule(repoSyncPeriod, false, func() {
		stats.loadRepos()
	})

	//schedules updates of latest versions
	commons.Schedule(versionsSyncPeriod, true, func() {
		stats.loadVersionsMap(includeBeta)
	})

	//schedules updates of commit stats
	commons.Schedule(contributorStatsSyncPeriod, true, func() {
		stats.loadContributionStats()
	})

	return stats
}

func (s *GitHubAggregator) loadContributionStats() {

	commitStats := make(map[StatRange]int, len(ranges))
	uniqueContributors := make(map[StatRange]int, len(ranges))

	for _, repo := range s.getRepos() {
		commons.Retry(statsRetryAttempts, statsRetryPeriod, func() error {
			stats, _, err := s.c.Repositories.ListContributorsStats(context.Background(), rpOrg, repo.GetName())
			if nil != err {
				log.Errorf("[%s] : %s", repo.GetName(), err.Error())
				return err
			}

			//summ statistics per range
			for _, tr := range ranges {
				rangeCount := 0

				//collect data of each contributor
				for _, contributorStats := range stats {
					contributorRangeCount := 0
					for _, weekStat := range contributorStats.Weeks[len(contributorStats.Weeks)-int(tr):] {
						contributorRangeCount += weekStat.GetCommits()
					}
					if contributorRangeCount > 0 {
						uniqueContributors[tr] = uniqueContributors[tr] + 1
					}
					rangeCount += contributorRangeCount
					log.Infof("%s for %d weeks: %d", contributorStats.Author.GetLogin(), tr, contributorRangeCount)
				}
				commitStats[tr] = commitStats[tr] + rangeCount
			}

			return nil
		})

	}

	s.contributionStats = &ContributionStats{
		Commits:      commitStats,
		Contributors: uniqueContributors,
	}

}

func (s *GitHubAggregator) loadVersionsMap(includeBeta bool) {
	repos := s.getRepos()
	versionMap := make(map[string]string, len(repos))

	for _, repo := range repos {
		var tagsRs []*github.RepositoryTag
		rq, _ := sling.New().Get(repo.GetTagsURL()).Request()

		_, err := s.c.Do(context.Background(), rq, &tagsRs)
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

	s.ltmu.Lock()
	s.latestTags = versionMap
	s.ltmu.Unlock()
}

//getRepos returns copy of cached Github Repos
func (s *GitHubAggregator) getRepos() []*github.Repository {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	repos := make([]*github.Repository, len(s.repos))
	copy(repos, s.repos)
	return repos
}

//loadRepos loads repositories from GitHUB
func (s *GitHubAggregator) loadRepos() {
	opt := &github.RepositoryListByOrgOptions{Type: "all"}
	repos, _, err := s.c.Repositories.ListByOrg(context.Background(), rpOrg, opt)
	if nil == err {
		s.rmu.Lock()
		defer s.rmu.Unlock()
		s.repos = repos

	}
}

//GetLatestTags returns copy of latest versions/tags map
func (s *GitHubAggregator) GetLatestTags() map[string]string {
	s.ltmu.RLock()
	defer s.ltmu.RUnlock()
	return s.latestTags
}

//GetStars returns count of stars for each repository and total count
func (s *GitHubAggregator) GetStars() *Stars {
	repoStars := make(map[string]int, len(s.repos))
	total := 0
	for _, repo := range s.getRepos() {
		repoStars[repo.GetName()] = repo.GetStargazersCount()
		total += repo.GetStargazersCount()
	}
	return &Stars{Total: total, Repos: repoStars}
}

//GetContributionStats returns aggregated contribution stats for organization repositories
func (s *GitHubAggregator) GetContributionStats() *ContributionStats {
	s.csmu.RLock()
	defer s.csmu.RUnlock()

	return s.contributionStats
}
