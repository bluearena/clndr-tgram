package main

import (
	"io/ioutil"
	"log"

	"github.com/yanzay/tbot"
)

func main() {

	token := getToken()

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
	message.Reply("okay")
}

func DeleteTaskHandler(message *tbot.Message) {
	message.Reply("okay")
}

func EditTaskHandler(message *tbot.Message) {
	message.Reply("okay")
}

func ShowTasksHandler(message *tbot.Message) {
	message.Reply("okay")
}

func getToken() string {
	s, err := ioutil.ReadFile("bot_token.config")
	checkError(err)
	return string(s)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
