package akane

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
)

func containNoticeText(text string) bool {
	keywords := []string{"配信", "時から", "分から", "showroom", "youtube", "live", "放送開始", "ニコニコ", "視聴", "ラジオ", "放送"}
	for _, k := range keywords {
		if strings.Contains(strings.ToLower(text), k) {
			return true
		}
	}
	return false
}

func HTTPFunction(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ENV_PRODUCTION") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	consumerKey := os.Getenv("CONSUMER_KEY")
	consumerSecretKey := os.Getenv("CONSUMER_SECRET_KEY")
	accessToken := os.Getenv("ACCESS_TOKEN")
	accessTokenSecret := os.Getenv("ACCESS_TOKEN_SECRET")

	config := oauth1.NewConfig(consumerKey, consumerSecretKey)
	token := oauth1.NewToken(accessToken, accessTokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	client := twitter.NewClient(httpClient)
	members, resp, err := client.Lists.Members(&twitter.ListsMembersParams{ListID: 1375104607026749440})
	if err != nil {
		log.Fatalf("Error getting members %v", err)
	}
	if resp.StatusCode == 200 {
		for _, user := range members.Users {
			tweets, resp, err := client.Timelines.UserTimeline(&twitter.UserTimelineParams{
				UserID:          user.ID,
				TrimUser:        twitter.Bool(false),
				ExcludeReplies:  twitter.Bool(true),
				IncludeRetweets: twitter.Bool(false),
			})
			if err != nil {
				log.Fatalf("Error getting statuses %v", err)
			}
			if resp.StatusCode == 200 {
				for _, tweet := range tweets {
					if containNoticeText(tweet.Text) {
						fmt.Printf("https://twitter.com/%s/status/%d\n", tweet.User.ScreenName, tweet.ID)
					}
				}
			}

		}
	}
}