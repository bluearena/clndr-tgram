package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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
var bot *tbot.Server

func main() {

	// get the telegram bot token, the google calendar client and the calendar ID
	token := getToken()
	google_client := getGoogleClient()
	calendarId = getCalendarID()

	//initialize the service for the calendar
	var err error
	srv, err = calendar.New(google_client)
	checkError(err)

	bot, err = tbot.NewServer(token) //create new server with /help defaulted
	checkError(err)

	//run StartHandler if /start command is received
	bot.HandleFunc("/start", startHandler)
	bot.HandleFunc("/add {eventstring}", CreateTaskHandler)
	bot.HandleFunc("/delete {eventstring}", DeleteTaskHandler)
	bot.HandleFunc("/show {number}", ShowTasksHandler)
	bot.HandleFunc("/todo", TodoHandler)

	log.Println("Starting Bot..")
	bot.ListenAndServe() //start server

}

func startHandler(message *tbot.Message) {
	//initialize the available buttons after /start
	// buttons := [][]string{
	// 	{"Termin erstellen", "Termin löschen"},
	// 	{"Termin bearbeiten", "Termine anzeigen"},
	// }
	// //show the buttons
	// message.ReplyKeyboard("Was kann ich für dich tun?", buttons)
	// message.ReplyKeyboard(text, buttons)
}

func CreateTaskHandler(message *tbot.Message) {
	user_input := message.Vars["eventstring"]
	splitted_string := strings.Split(user_input, " ")
	event_name := splitted_string[0]
	splitted_time := strings.Split(splitted_string[1], "-")
	date := splitted_string[2]

	parsed_stime, _ := time.Parse("15:04 02/01/2006", splitted_time[0]+" "+date)
	parsed_etime, _ := time.Parse("15:04 02/01/2006", splitted_time[1]+" "+date)
	formatted_stime := parsed_stime.Format(time.RFC3339)
	formatted_etime := parsed_etime.Format(time.RFC3339)

	//adjust Time entries from UTC to UTC+1
	start := &calendar.EventDateTime{
		DateTime: strings.Replace(formatted_stime, "Z", "", len(formatted_stime)) + "+01:00",
		TimeZone: "Europe/Berlin",
	}
	end := &calendar.EventDateTime{
		DateTime: strings.Replace(formatted_etime, "Z", "", len(formatted_stime)) + "+01:00",
		TimeZone: "Europe/Berlin",
	}

	evt := &calendar.Event{Summary: event_name, Start: start, End: end}
	_, err := srv.Events.Insert(calendarId, evt).Do()
	checkError(err)
	reply := fmt.Sprintf("Termin %v (%v %v) hinzugefügt", event_name, date, splitted_string[1])
	message.Reply(reply)
}

func DeleteTaskHandler(message *tbot.Message) {
	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(calendarId).ShowDeleted(false).SingleEvents(true).TimeMin(t).MaxResults(200).OrderBy("startTime").Do()
	checkError(err)

	deleteNumber, err := strconv.Atoi(message.Vars["eventstring"])
	checkError(err)
	eventId := events.Items[deleteNumber-1].Id
	event_name := events.Items[deleteNumber-1].Summary

	srv.Events.Delete(calendarId, eventId).Do()

	reply := fmt.Sprintf("Termin %v gelöscht", event_name)
	message.Reply(reply)
}

func ShowTasksHandler(message *tbot.Message) {
	number_results_string := message.Vars["number"]
	number_results, err := strconv.ParseInt(number_results_string, 10, 64)
	checkError(err)

	if message.Vars["number"] == "" {
		number_results = 100
	}

	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(calendarId).ShowDeleted(false).SingleEvents(true).TimeMin(t).MaxResults(number_results).OrderBy("startTime").Do()
	checkError(err)
	var formattedEvents string

	if len(events.Items) == 0 {
		message.Reply("Keine anstehenden Termine.")
	} else {
		formattedEvents += "Die nächsten " + number_results_string + " Termine: \n\n"
		for i, item := range events.Items {
			date := item.Start.DateTime
			parsed_time, _ := time.Parse(time.RFC3339, date)

			end_date := item.End.DateTime
			parsed_end_date, _ := time.Parse(time.RFC3339, end_date)
			formatted_end_date := parsed_end_date.Format("15:04")

			if date == "" {
				date = item.Start.Date
				parsed_time, _ = time.Parse("2006-01-02", date)
				formatted_end_date = "00:00"
			}

			formatted_date := parsed_time.Format("02/01/2006 15:04")

			event_string := fmt.Sprintf("%v (%v-%v)\n", item.Summary, formatted_date, formatted_end_date)
			formattedEvents += "[" + strconv.Itoa(i+1) + "] " + event_string
		}
		message.Reply(formattedEvents)
	}
}

func TodoHandler(message *tbot.Message) {

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
