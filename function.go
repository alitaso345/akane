package akane

import (
	"context"
	json2 "encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/iterator"

	firebase "firebase.google.com/go/v4"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/linebot"
	"google.golang.org/api/option"
)

type User struct {
	LineId string `json:"lineId"`
}

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
	members, resp, err := client.Lists.Members(&twitter.ListsMembersParams{ListID: 1375784169129738240})
	if err != nil {
		log.Fatalf("Error getting members %v", err)
	}

	var text string
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
					if containNoticeText(tweet.Text) && isRecentTweet(tweet) {
						text = text + "https://twitter.com/" + tweet.User.ScreenName + "/status/" + strconv.FormatInt(tweet.ID, 10) + "\n\n"

					}
				}
			}

		}
	}

	userIds := getUserIds()
	for _, id := range userIds {
		sendLineMessage(id, text)
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

	var app *firebase.App
	ctx := context.Background()
	if os.Getenv("ENV") != "production" {
		app, _ = firebase.NewApp(ctx, nil, option.WithCredentialsFile("serviceAccount.json"))
	} else {
		app, _ = firebase.NewApp(ctx, nil)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

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

	for _, event := range events {
		if event.Type == linebot.EventTypeFollow {
			_, err := client.Collection("users").Doc(event.Source.UserID).Set(ctx, map[string]interface{}{
				"lineId": event.Source.UserID,
			})
			if err != nil {
				log.Fatalf("Failed adding alovelace: %v", err)
			}
		}

		if event.Type == linebot.EventTypeUnfollow {
			doc := client.Collection("users").Doc(event.Source.UserID)
			log.Printf("AKANE: doc %v", doc)
			_, err := doc.Delete(ctx)
			if err != nil {
				log.Fatalf("An error has occurred: %s\", err", err)
			}
		}
	}
	fmt.Fprintf(w, "ok")
}

func getUserIds() []string {
	// デバッグ用
	//return []string{os.Getenv("MY_USER_ID")}

	var userIds []string

	var app *firebase.App
	ctx := context.Background()
	if os.Getenv("ENV") != "production" {
		app, _ = firebase.NewApp(ctx, nil, option.WithCredentialsFile("serviceAccount.json"))
	} else {
		app, _ = firebase.NewApp(ctx, nil)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	iter := client.Collection("users").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		json, err := json2.Marshal(doc.Data())
		if err != nil {
			log.Fatalln(err)
		}
		var user User
		json2.Unmarshal(json, &user)
		userIds = append(userIds, user.LineId)
	}
	return userIds
}

func sendLineMessage(userId string, message string) {
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	channelSecret := os.Getenv("CHANNEL_SECRET")
	channelAccessToken := os.Getenv("CHANNEL_ACCESS_TOKEN")

	client := &http.Client{}
	bot, err := linebot.New(channelSecret, channelAccessToken, linebot.WithHTTPClient(client))
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var messages []linebot.SendingMessage
	messages = append(messages, linebot.NewTextMessage(message))
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

func isRecentTweet(tweet twitter.Tweet) bool {
	now := time.Now()
	createdAt, _ := time.Parse(time.RubyDate, tweet.CreatedAt)
	return int(now.Sub(createdAt).Hours())/24 == 0
}
