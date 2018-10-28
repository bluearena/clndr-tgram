package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/yanzay/tbot"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

/* global variables */
var srv *calendar.Service
var calendarId string

func main() {

	// get the telegram bot token, the google calendar client and the calendar ID
	token := getToken()
	google_client := getGoogleClient()
	calendarId = getCalendarID()

	//initialize the service for the calendar
	var err error
	srv, err = calendar.New(google_client)
	checkError(err)

	bot, err := tbot.NewServer(token) //create new server with /help defaulted
	checkError(err)

	//run StartHandler if /start command is received
	bot.HandleFunc("/start", startHandler)

	//handle the according button press after /start command
	bot.HandleFunc("Termin erstellen", CreateTaskHandler)
	bot.HandleFunc("Termin löschen", DeleteTaskHandler)
	bot.HandleFunc("Termin bearbeiten", EditTaskHandler)
	bot.HandleFunc("Termine anzeigen", ShowTasksHandler)

	log.Println("Starting Bot..")
	bot.ListenAndServe() //start server
}

func startHandler(message *tbot.Message) {
	//initialize the available buttons after /start
	buttons := [][]string{
		{"Termin erstellen", "Termin löschen"},
		{"Termin bearbeiten", "Termine anzeigen"},
	}
	//show the buttons
	message.ReplyKeyboard("Was kann ich für dich tun?", buttons)
}

func CreateTaskHandler(message *tbot.Message) {
	//TODO
	//srv.Events.Insert(calendarId, event)
}

func DeleteTaskHandler(message *tbot.Message) {
	message.Reply("okay")
}

func EditTaskHandler(message *tbot.Message) {
	message.Reply("okay")
}

func ShowTasksHandler(message *tbot.Message) {
	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(calendarId).ShowDeleted(false).SingleEvents(true).TimeMin(t).MaxResults(20).Do()
	checkError(err)
	var formattedEvents string

	if len(events.Items) == 0 {
		message.Reply("Keine anstehenden Termine.")
	} else {
		formattedEvents += "Die nächsten Termine (bis zu 20): \n\n"
		for _, item := range events.Items {
			date := item.Start.DateTime
			log.Println(reflect.TypeOf(date))
			if date == "" {
				date = item.Start.Date
			}
			event_string := fmt.Sprintf("%v (%v)\n", item.Summary, date)
			formattedEvents += "- " + event_string
		}
	}

	message.Reply(formattedEvents)

}

func getToken() string {
	s, err := ioutil.ReadFile("bot_token.config")
	checkError(err)
	return string(s)
}

func getCalendarID() string {
	s, err := ioutil.ReadFile("calendar_id")
	checkError(err)
	return string(s)
}

func getGoogleClient() *http.Client {
	//read the credentials
	s, err := ioutil.ReadFile("credentials.json")
	checkError(err)

	//get the rights to view and edit calendar events
	config, err := google.ConfigFromJSON(s, calendar.CalendarEventsScope)
	checkError(err)

	client := getClient(config)

	return client
}

//retrieve a token, save the token, then return the generated client
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)

	//if no token File is present, get a token from the web and save it to the token file
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}

	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
