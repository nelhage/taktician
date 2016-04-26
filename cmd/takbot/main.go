package main

import (
	"flag"
	"log"

	"nelhage.com/tak/ai"
)

var (
	server = flag.String("server", "playtak.com:10000", "playtak.com server to connect to")
	depth  = flag.Int("depth", 5, "minimax depth")
	user   = flag.String("user", "", "username for login")
	pass   = flag.String("pass", "", "password for login")
)

const Client = "Takker AI"

func main() {
	flag.Parse()
	client := &client{
		ai:    ai.NewMinimax(*depth),
		debug: true,
	}
	err := client.Connect(*server)
	if err != nil {
		log.Fatal(err)
	}
	client.SendClient(Client)
	if *user != "" {
		err = client.Login(*user, *pass)
	} else {
		err = client.LoginGuest()
	}
	if err != nil {
		log.Fatal("login: ", err)
	}
}
