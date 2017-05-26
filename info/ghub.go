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
	rpOrg string = "reportportal"
	//open/closed pr/issue
	issueQueryTemplate string = "is:%s is:%s user:" + rpOrg

	repoSyncPeriod             time.Duration = time.Hour * 4
	versionsSyncPeriod         time.Duration = time.Hour
	contributorStatsSyncPeriod time.Duration = time.Hour * 12
	commitsStatsSyncPeriod     time.Duration = time.Hour * 6
	issuesStatsSyncPeriod      time.Duration = time.Minute * 30
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

	csmu               *sync.RWMutex
	commitStats        map[StatRange]int
	uniqueContributors map[StatRange]int

	ismu       *sync.RWMutex
	issueStats IssueStats
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

//IssueStats hold issues stats
type IssueStats struct {
	OpenPRs      int `json:"open_pull_requests"`
	OpenIssues   int `json:"open_issues"`
	ClosedIssues int `json:"closed_issues"`
	TotalIssues  int `json:"total_issues"`
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
		ismu: &sync.RWMutex{},
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
		stats.loadUniqueContributors()
	})

	//schedules updates of commit stats
	commons.Schedule(commitsStatsSyncPeriod, true, func() {
		stats.loadCommitStats()
	})

	//schedules updates of issue stats
	commons.Schedule(issuesStatsSyncPeriod, true, func() {
		stats.loadIssueStats()
	})

	return stats
}

func (s *GitHubAggregator) loadCommitStats() {
	commitStats := make(map[StatRange]int, len(ranges))

	for _, repo := range s.getRepos() {
		commons.Retry(5, time.Second*10, func() error {
			stats, _, err := s.c.Repositories.ListCommitActivity(context.Background(), rpOrg, repo.GetName())
			if nil != err {
				log.Errorf("[%s] : %s", repo.GetName(), err.Error())
				return err
			}

			for _, tr := range ranges {
				count := 0
				for _, stat := range stats[len(stats)-int(tr):] {
					count += stat.GetTotal()
				}
				commitStats[tr] += count
			}
			return nil
		})
	}
	s.csmu.Lock()
	s.commitStats = commitStats
	s.csmu.Unlock()
}
func (s *GitHubAggregator) loadUniqueContributors() {

	uniqueContributors := make(map[StatRange]int, len(ranges))

	for _, repo := range s.getRepos() {
		commons.Retry(statsRetryAttempts, statsRetryPeriod, func() error {
			contributors, _, err := s.c.Repositories.ListContributorsStats(context.Background(), rpOrg, repo.GetName())
			if nil != err {
				log.Debugf("[%s] : %s", repo.GetName(), err.Error())
				return err
			}

			//summ statistics per range
			for _, tr := range ranges {
				//collect data of each contributor
				for _, contributor := range contributors {
					for _, weekStat := range contributor.Weeks[len(contributor.Weeks)-int(tr):] {
						if weekStat.GetCommits() > 0 {
							uniqueContributors[tr]++
							break
						}
					}

				}
			}

			return nil
		})

	}

	s.csmu.Lock()
	s.uniqueContributors = uniqueContributors
	s.csmu.Unlock()

}

//loadVersionsMap loads the latest tags
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

//loadIssueStats loads issue statistics
func (s *GitHubAggregator) loadIssueStats() {
	prs, _, err := s.c.Search.Issues(context.Background(), fmt.Sprintf(issueQueryTemplate, "open", "pr"), &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}})
	if nil != err {
		log.Errorf("Unable to find PRs count. %s", err.Error())
	}

	issues, _, err := s.c.Search.Issues(context.Background(), fmt.Sprintf(issueQueryTemplate, "open", "issue"), &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}})
	if nil != err {
		log.Errorf("Unable to find PRs count. %s", err.Error())
	}

	closedIssues, _, err := s.c.Search.Issues(context.Background(), fmt.Sprintf(issueQueryTemplate, "closed", "issue"), &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}})
	if nil != err {
		log.Errorf("Unable to find PRs count. %s", err.Error())
	}

	s.ismu.Lock()
	defer s.ismu.Unlock()
	s.issueStats = IssueStats{
		OpenPRs:      prs.GetTotal(),
		OpenIssues:   issues.GetTotal(),
		ClosedIssues: closedIssues.GetTotal(),
		TotalIssues:  issues.GetTotal() + closedIssues.GetTotal(),
	}

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
	tags := make(map[string]string, len(s.latestTags))
	for k, v := range s.latestTags {
		tags[k] = v
	}
	return tags
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

	commits := make(map[StatRange]int, len(s.commitStats))
	for k, v := range s.commitStats {
		commits[k] = v
	}

	contributors := make(map[StatRange]int, len(s.uniqueContributors))
	for k, v := range s.uniqueContributors {
		contributors[k] = v
	}

	return &ContributionStats{Commits: commits, Contributors: contributors}
}

//GetIssueStats returns issues/PRs statistics
func (s *GitHubAggregator) GetIssueStats() IssueStats {
	s.ismu.RLock()
	defer s.ismu.RUnlock()
	return s.issueStats
}
