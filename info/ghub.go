package info

import (
	"context"
	"fmt"
	"github.com/dghubble/sling"
	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

const rpOrg = "reportportal"

//GitHubVersions is a structure for retrieving DockerHub tags
type GitHubVersions struct {
	repoLatest  map[string]string
	includeBeta bool
	mu          *sync.RWMutex
	client      *github.Client
	ctx         context.Context
}

//NewGitHubVersions creates new struct with default values
func NewGitHubVersions(ghToken string, includeBeta bool) *GitHubVersions {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	versions := &GitHubVersions{
		mu:          &sync.RWMutex{},
		client:      github.NewClient(tc),
		includeBeta: includeBeta,
		ctx:         ctx}

	//schedules updates of latest versions
	duration := time.Hour
	ticker := time.Tick(duration)

	go func() {
		//initially loads the latest versions
		versions.load()
		for range ticker {
			versions.load()
		}
	}()

	return versions
}

func (v *GitHubVersions) load() {
	v.mu.Lock()
	defer v.mu.Unlock()

	versionMap := map[string]string{}

	opt := &github.RepositoryListByOrgOptions{Type: "all"}
	repos, _, err := v.client.Repositories.ListByOrg(v.ctx, rpOrg, opt)
	if nil != err {
		log.Println(err)
		return
	}
	for _, repo := range repos {
		var tagsRs []*github.RepositoryTag
		_, err := sling.New().Get(repo.GetTagsURL()).ReceiveSuccess(&tagsRs)
		if nil != err {
			log.Println(err)
			return
		}
		versions := version.Collection([]*version.Version{})
		for _, tag := range tagsRs {
			name := tag.GetName()

			//not a latest (we need explicit version), not a beta
			if "" != name && (v.includeBeta || !strings.Contains(strings.ToLower(name), "beta")) {
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
	v.repoLatest = versionMap
}

//Do executes provided callback on latest versions/tags map
func (v *GitHubVersions) Do(f func(map[string]string)) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	f(v.repoLatest)
}
