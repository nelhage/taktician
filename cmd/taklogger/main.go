package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"nelhage.com/tak/playtak"
	"nelhage.com/tak/ptn"
	"nelhage.com/tak/tak"
)

var (
	server = flag.String("server", "playtak.com:10000", "playtak.com server to connect to")
	out    = flag.String("out", "ptn", "Directory to write PTN files")
)

const ClientName = "Taktician Logger"

func main() {
	flag.Parse()
	client := &playtak.Client{
		Debug: true,
	}
	err := client.Connect(*server)
	if err != nil {
		log.Fatal(err)
	}
	client.SendClient(ClientName)
	err = client.LoginGuest()
	if err != nil {
		log.Fatal("login: ", err)
	}

	if e := logGames(client, *out); e != nil {
		log.Fatal(e)
	}
}

type Game struct {
	Id     string
	White  string
	Black  string
	Date   string
	Site   string
	Size   int
	Moves  []tak.Move
	Result string
}

func logGames(client *playtak.Client, out string) error {
	e := os.MkdirAll(out, 0755)
	if e != nil {
		return e
	}
	games := make(map[string]*Game)
	for line := range client.Recv {
		if strings.HasPrefix(line, "GameList Add") {
			if g := addGame(games, line); g != nil {
				client.SendCommand("Observe", g.Id)
			}
		}
		if !strings.HasPrefix(line, "Game#") {
			continue
		}
		words := strings.Split(line, " ")
		no := strings.SplitN(words[0], "#", 2)[1]
		if g, ok := games[no]; ok {
			if over := handleCmd(g, words); over {
				render(g, out)
				delete(games, no)
			}
		}
	}
	return nil
}

func addGame(games map[string]*Game, line string) *Game {
	words := strings.Split(line, " ")
	no := strings.SplitN(words[2], "#", 2)[1]
	g := &Game{
		Id:    no,
		White: words[3],
		Black: strings.TrimRight(words[5], ","),
		Size:  int(words[6][0] - '0'),
		Date:  time.Now().Format("2006-01-02"),
		Site:  *server,
	}
	games[no] = g
	return g
}

func handleCmd(g *Game, cmd []string) bool {
	switch cmd[1] {
	case "P", "M":
		m, e := playtak.ParseServer(strings.Join(cmd[1:], " "))
		if e != nil {
			log.Printf("bad move: %v", cmd)
			return true
		}
		g.Moves = append(g.Moves, m)
	case "Undo":
		g.Moves = g.Moves[:len(g.Moves)-1]
	case "Over":
		g.Result = cmd[2]
		return true
	case "Abandoned.":
	}
	return false
}

func render(g *Game, dir string) {
	p := ptn.PTN{}
	p.Tags = []ptn.Tag{
		{Name: "Size", Value: strconv.Itoa(g.Size)},
		{Name: "Date", Value: g.Date},
		{Name: "Player1", Value: g.White},
		{Name: "Player2", Value: g.Black},
		{Name: "Result", Value: g.Result},
	}
	for i, m := range g.Moves {
		if i%2 == 0 {
			p.Ops = append(p.Ops, &ptn.MoveNumber{Number: i/2 + 1})
		}
		p.Ops = append(p.Ops, &ptn.Move{Move: m})
	}
	out := p.Render()
	e := ioutil.WriteFile(fmt.Sprintf("%s/%s.ptn", dir, g.Id),
		[]byte(out), 0644)
	if e != nil {
		log.Printf("write game: %v", e)
	}
}
