package info

import (
	"context"
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

//GitHubTags is a structure for retrieving DockerHub tags
type GitHubTags struct {
	repoLatest  map[string]string
	includeBeta bool
	mu          *sync.RWMutex
	client      *github.Client
	ctx         context.Context
}

//NewGitHubTags creates new struct with default values
func NewGitHubTags(includeBeta bool) *GitHubTags {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "e27ff5b9926e7a5abaa17410bd4fbebac26fb4d6"},
	)
	tc := oauth2.NewClient(ctx, ts)

	versions := &GitHubTags{
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

func (v *GitHubTags) load() {
	v.mu.Lock()
	defer v.mu.Unlock()

	versionMap := map[string]string{}

	opt := &github.RepositoryListByOrgOptions{Type: "all"}
	repos, _, err := v.client.Repositories.ListByOrg(v.ctx, "reportportal", opt)
	if nil != err {
		log.Println(err)
		return
	}
	for _, repo := range repos {
		var tagsRs []*github.RepositoryTag
		rq, _ := sling.New().Get(repo.GetTagsURL()).Request()
		_, err := v.client.Do(v.ctx, rq, &tagsRs)
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
		versionMap[repo.GetName()] = versions[len(versions)-1].String()
	}
	v.repoLatest = versionMap
}

//Do executes provided callback on latest versions/tags map
func (v *GitHubTags) Do(f func(map[string]string)) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	f(v.repoLatest)
}
