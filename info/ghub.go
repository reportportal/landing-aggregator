package info

import (
	"context"
	"fmt"
	"github.com/dghubble/sling"
	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/reportportal/commons-go.v1/commons"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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

	repos              atomic.Value
	latestTags         atomic.Value
	commitStats        atomic.Value
	uniqueContributors atomic.Value
	issueStats         atomic.Value
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
		c:                  ghClient,
		latestTags:         atomic.Value{},
		repos:              atomic.Value{},
		commitStats:        atomic.Value{},
		uniqueContributors: atomic.Value{},
		issueStats:         atomic.Value{},
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
	log.Debugf("Updating commit statistics...")

	commitStats := make(map[StatRange]int)

	mu := sync.Mutex{}
	s.doWithRepos(func(repo *github.Repository) {
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
				mu.Lock()
				commitStats[tr] += count
				mu.Unlock()
			}
			return nil
		})
	})

	s.commitStats.Store(commitStats)
}
func (s *GitHubAggregator) loadUniqueContributors() {
	log.Debugf("Updating unique contributors set...")

	mu := sync.Mutex{}
	uniqueContributors := make(map[StatRange]int, len(ranges))

	s.doWithRepos(func(repo *github.Repository) {
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
					var weeklyStats []github.WeeklyStats

					if len(contributor.Weeks) > int(tr) {
						weeklyStats = contributor.Weeks[len(contributor.Weeks)-int(tr):]
					} else {
						weeklyStats = contributor.Weeks
					}

					for _, weekStat := range weeklyStats {
						if weekStat.GetCommits() > 0 {
							mu.Lock()
							uniqueContributors[tr]++
							mu.Unlock()
							break
						}
					}

				}
			}

			return nil
		})
	})

	s.uniqueContributors.Store(uniqueContributors)

}

//loadVersionsMap loads the latest tags
func (s *GitHubAggregator) loadVersionsMap(includeBeta bool) {
	log.Debugf("Updating latest versions map...")

	mu := sync.Mutex{}
	versionMap := make(map[string]string)

	s.doWithRepos(func(repo *github.Repository) {
		var tagsRs []*github.RepositoryTag
		rq, err := sling.New().Get(repo.GetTagsURL()).Request()
		if nil != err {
			log.Error(err)
			return
		}

		if _, err = s.c.Do(context.Background(), rq, &tagsRs); nil != err {
			log.Error(err)
			return
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
			mu.Lock()
			versionMap[fmt.Sprintf("%s/%s", rpOrg, repo.GetName())] = versions[len(versions)-1].String()
			mu.Unlock()
		} else {
			log.Debugf("Repo '%s' does not have valid version tags", repo.GetName())
		}
	})

	s.latestTags.Store(versionMap)
}

//loadIssueStats loads issue statistics
func (s *GitHubAggregator) loadIssueStats() {
	log.Debugf("Updating issue statistics...")

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

	s.issueStats.Store(&IssueStats{
		OpenPRs:      prs.GetTotal(),
		OpenIssues:   issues.GetTotal(),
		ClosedIssues: closedIssues.GetTotal(),
		TotalIssues:  issues.GetTotal() + closedIssues.GetTotal()})
}

//doWithRepos performs some action under cached repos in parallel manner
func (s *GitHubAggregator) doWithRepos(f func(repo *github.Repository)) {
	repos := s.repos.Load().([]*github.Repository)
	wg := sync.WaitGroup{}
	wg.Add(len(repos))
	for _, repo := range repos {
		go func(repo *github.Repository) {
			defer wg.Done()
			f(repo)
		}(repo)
	}
	wg.Wait()
}

//loadRepos loads repositories from GitHUB
func (s *GitHubAggregator) loadRepos() {
	opt := &github.RepositoryListByOrgOptions{Type: "all", ListOptions: github.ListOptions{PerPage: 50}}

	// get all pages of results
	var allRepos []*github.Repository
	for {
		repos, resp, err := s.c.Repositories.ListByOrg(context.Background(), rpOrg, opt)
		if err != nil {
			log.Errorf("Cannot get repositories list: ma%v", err)
			continue
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	log.Infof("%d repositories found", len(allRepos))
	s.repos.Store(allRepos)

}

//GetLatestTags returns copy of latest versions/tags map
func (s *GitHubAggregator) GetLatestTags() map[string]string {
	return s.latestTags.Load().(map[string]string)
}

//GetStars returns count of stars for each repository and total count
func (s *GitHubAggregator) GetStars() *Stars {
	total := 0
	repos := s.repos.Load().([]*github.Repository)

	repoStars := make(map[string]int, len(repos))
	for _, repo := range repos {
		repoStars[repo.GetName()] = repo.GetStargazersCount()
		total += repo.GetStargazersCount()
	}
	return &Stars{Total: total, Repos: repoStars}
}

//GetContributionStats returns aggregated contribution stats for organization repositories
func (s *GitHubAggregator) GetContributionStats() *ContributionStats {
	return &ContributionStats{Commits: s.commitStats.Load().(map[StatRange]int), Contributors: s.uniqueContributors.Load().(map[StatRange]int)}
}

//GetIssueStats returns issues/PRs statistics
func (s *GitHubAggregator) GetIssueStats() *IssueStats {
	return s.issueStats.Load().(*IssueStats)
}
