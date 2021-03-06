package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
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
var (
	srv        *calendar.Service
	calendarId string
	bot        *tbot.Server
)

func main() {

	// get the telegram bot token, the google calendar client and the calendar ID
	token := os.Getenv("BOTTOKEN")
	calendarId = os.Getenv("CALENDARID")
	google_client := getGoogleClient()

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
	bot.HandleFunc("/show", ShowTasksHandler)
	bot.HandleFunc("/todo", TodoHandler)

	log.Println("Starting Bot..")
	bot.ListenAndServe() //start server

}

func startHandler(message *tbot.Message) {

}

func CreateTaskHandler(message *tbot.Message) {
	user_input := message.Vars["eventstring"]

	//get the whole date in format dd/mm/yyyy or d/m/yyyy, dd/m/yyyy, d/m/yyyy
	date_expr := regexp.MustCompile("(0?[1-9]|[12][0-9]|3[01])/(0?[1-9]|1[012])/((19|20)\\d\\d)")
	date := date_expr.FindString(user_input)

	//get the time in format 10:00-12:00
	time_expr := regexp.MustCompile("([01]?[0-9]|2[0-3]):[0-5][0-9]-([01]?[0-9]|2[0-3]):[0-5][0-9]")
	compl_time := time_expr.FindString(user_input)

	//get the start and end time in format 10:00
	time_slice_expr := regexp.MustCompile("([01]?[0-9]|2[0-3]):[0-5][0-9]")
	time_slice := time_slice_expr.FindAllString(compl_time, -1)
	stime := time_slice[0]
	etime := time_slice[1]

	//remove date and time from the user input. So we only have the name of the event left
	//additionally trim trailing and leading spaces from the name
	name := strings.Trim(strings.Replace(strings.Replace(user_input, compl_time, "", -1), date, "", -1), " ")

	//adjust time formats to RFC3339 since the calendar API expects it in that format
	parsed_stime, _ := time.Parse("15:04 02/01/2006", stime+" "+date)
	parsed_etime, _ := time.Parse("15:04 02/01/2006", etime+" "+date)
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

	//add the event to the calendar
	evt := &calendar.Event{Summary: name, Start: start, End: end}
	_, err := srv.Events.Insert(calendarId, evt).Do()
	checkError(err)
	reply := fmt.Sprintf("Termin %v (%v %v) hinzugefügt", name, date, compl_time)
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
	var number_results int64
	var err error

	if message.Vars["number"] == "" {
		number_results = 100
	} else {
		number_results, err = strconv.ParseInt(message.Vars["number"], 10, 64)
		checkError(err)
	}

	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(calendarId).ShowDeleted(false).SingleEvents(true).TimeMin(t).MaxResults(number_results).OrderBy("startTime").Do()
	checkError(err)
	var formattedEvents string

	if len(events.Items) == 0 {
		message.Reply("Keine anstehenden Termine.")
	} else {
		formattedEvents += "Die nächsten " + strconv.FormatInt(number_results, 10) + " Termine: \n\n"
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

func getGoogleClient() *http.Client {
	clientid := os.Getenv("CLIENTID")
	clientsecret := os.Getenv("CLIENTSECRET")
	redirecturl := os.Getenv("REDIRECTURL")

	conf := &oauth2.Config{
		ClientID:     clientid,
		ClientSecret: clientsecret,
		RedirectURL:  redirecturl,
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar",
		},
		Endpoint: google.Endpoint,
	}

	authURL := conf.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	log.Println("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)
	authCode := os.Getenv("AUTHCODE")
	tok, err := conf.Exchange(context.TODO(), authCode)
	checkError(err)

	client := conf.Client(context.Background(), tok)

	return client
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
