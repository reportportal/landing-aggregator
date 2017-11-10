package main

import (
	"github.com/reportportal/landing-aggregator/info"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

func main() {
	key, err := ioutil.ReadFile("reportportal-eaef53789908.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	//"UCsZxrHqLHPJcrkcgIGRG-cQ"
	videos, err := info.NewYoutubeVideosBuffer("UCsZxrHqLHPJcrkcgIGRG-cQ", 10, key)
	for i, video := range videos.GetVideos() {
		fmt.Printf("%d : %s\n", i, video)
	}
	time.Sleep(50 * time.Second)

	for i, video := range videos.GetVideos() {
		fmt.Printf("%d : %s\n", i, video)
	}

	//"snippet,contentDetails,statistics"
	//info.ChannelsListByUsername(serv, "snippet,contentDetails,statistics", "Report Portal Community")
}
