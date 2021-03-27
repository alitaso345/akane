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
	"github.com/line/line-bot-sdk-go/linebot"
)

func HTTPFunction(_w http.ResponseWriter, _r *http.Request) {
	if os.Getenv("ENV") != "production" {
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

func LineBotWebhookFunction(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}
	channelSecret := os.Getenv("CHANNEL_SECRET")
	channelAccessToken := os.Getenv("CHANNEL_ACCESS_TOKEN")

	bot, err := linebot.New(channelSecret, channelAccessToken)
	if err != nil {
		log.Fatal(err)
	}

	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	log.Println("Loading webhook function")
	for _, event := range events {
		if event.Type == linebot.EventTypeFollow {
			log.Println("Follow event")
			log.Println(event.Source.UserID)
		}

		if event.Type == linebot.EventTypeUnfollow {
			log.Println("Unfollow event")
			log.Println(event.Source.UserID)
		}
	}
	fmt.Fprintf(w, "ok")
}

func SendLINE(_w http.ResponseWriter, _r *http.Request) {
	fmt.Println("LINE BOT testing...")
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	channelSecret := os.Getenv("CHANNEL_SECRET")
	channelAccessToken := os.Getenv("CHANNEL_ACCESS_TOKEN")
	userId := os.Getenv("MY_USER_ID")

	client := &http.Client{}
	bot, err := linebot.New(channelSecret, channelAccessToken, linebot.WithHTTPClient(client))
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var messages []linebot.SendingMessage
	messages = append(messages, linebot.NewTextMessage("Hello"))
	_, err = bot.PushMessage(userId, messages...).Do()
	if err != nil {
		log.Fatal(err)
	}
}

func containNoticeText(text string) bool {
	keywords := []string{"配信", "時から", "分から", "showroom", "youtube", "live", "放送開始", "ニコニコ", "視聴", "ラジオ", "放送"}
	for _, k := range keywords {
		if strings.Contains(strings.ToLower(text), k) {
			return true
		}
	}
	return false
}
