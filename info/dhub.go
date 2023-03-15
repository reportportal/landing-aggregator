package info

import (
	"fmt"
	"github.com/dghubble/sling"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	dockerHubBase       = "https://registry.hub.docker.com"
	hubSearchTemplate   = "/v1/search?q=%s&page=1&n=25"
	hubRepoTagsTemplate = "/v1/repositories/%s/tags"
)

// DHubTags is a structure for retrieving DockerHub tags
type DHubTags struct {
	repoLatest  map[string]string
	includeBeta bool
	mu          *sync.RWMutex
	client      *sling.Sling
}

// NewDockerHubTags creates new struct with default values
func NewDockerHubTags(includeBeta bool) *DHubTags {
	versions := &DHubTags{
		mu: &sync.RWMutex{},
		client: sling.New().Base(dockerHubBase).Client(&http.Client{
			Timeout: time.Second * 10,
		}),
		includeBeta: includeBeta}

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

// Do executes provided callback on latest versions/tags map
func (v *DHubTags) Do(f func(map[string]string)) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	f(v.repoLatest)
}

func (v *DHubTags) findRepoNames() ([]string, error) {
	var orgRepos map[string]interface{}
	_, err := v.client.Get(fmt.Sprintf(hubSearchTemplate, "reportportal")).ReceiveSuccess(&orgRepos)
	if nil != err {
		return nil, err
	}

	repoDetails := orgRepos["results"].([]interface{})
	var repoNames = make([]string, len(repoDetails))

	for i, repo := range repoDetails {
		repoNames[i] = repo.(map[string]interface{})["name"].(string)
	}
	return repoNames, nil
}
func (v *DHubTags) load() {
	v.mu.Lock()
	defer v.mu.Unlock()

	versionMap := map[string]string{}

	names, err := v.findRepoNames()
	if nil != err {
		fmt.Println(err)
		return
	}

	for _, app := range names {
		var tags []map[string]string

		_, err := v.client.Get(fmt.Sprintf(hubRepoTagsTemplate, app)).ReceiveSuccess(&tags)
		if nil != err {
			continue
		}

		var versions []string
		for _, tag := range tags {
			name := tag["name"]
			//not a latest (we need explicit version), not a beta
			if "" != name && "latest" != name && (v.includeBeta || !strings.Contains(strings.ToLower(name), "beta")) {
				versions = append(versions, name)
			}
		}
		if 0 == len(versions) {
			continue
		}

		//sort a pick the latest one
		sort.Strings(versions)
		versionMap[app] = versions[len(versions)-1]
	}
	v.repoLatest = versionMap
}
