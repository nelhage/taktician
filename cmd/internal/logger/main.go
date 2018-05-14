package logger

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/logs"
	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
	server     string
	out        string
	index      string
	reindex    bool
	cpuProfile string
}

func (*Command) Name() string     { return "logger" }
func (*Command) Synopsis() string { return "Start a playtak.com game logger" }
func (*Command) Usage() string {
	return `logger [options]

Starts a playtak.com logger. Logs an index of games into a sqlite
database, and a directory of PTNs.
 `
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.StringVar(&c.server, "server", "playtak.com:10000", "playtak.com server to connect to")
	flags.StringVar(&c.out, "out", "ptn", "Directory to write PTN files")
	flags.StringVar(&c.index, "index", "", "write a sqlite index")
	flags.BoolVar(&c.reindex, "reindex", false, "reindex all games")
	flags.StringVar(&c.cpuProfile, "cpu-profile", "", "write a CPU profile")
}

const ClientName = "Taktician Logger"

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.cpuProfile != "" {
		f, e := os.OpenFile(c.cpuProfile, os.O_WRONLY|os.O_CREATE, 0644)
		if e != nil {
			log.Fatalf("open cpu-profile: %s: %v", c.cpuProfile, e)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if c.reindex {
		if c.index == "" {
			log.Fatal("-reindex requires -index")
		}
		os.Remove(c.index)
		repo, err := logs.Open(c.index)
		if err != nil {
			log.Fatal(err)
		}
		defer repo.Close()
		if err := indexPTN(repo, c.out, c.index); err != nil {
			log.Fatal(err)
		}
		return subcommands.ExitSuccess
	}

	var repo *logs.Repository
	if c.index != "" {
		var err error
		repo, err = logs.Open(c.index)
		if err != nil {
			log.Fatal(err)
		}
	}

	cl, err := playtak.Dial(true, c.server)
	if err != nil {
		log.Fatal(err)
	}
	client := &playtak.Commands{"", cl}
	client.SendClient(ClientName)
	err = client.LoginGuest()
	if err != nil {
		log.Fatal("login: ", err)
	}

	if e := logGames(repo, client, c.server, c.out); e != nil {
		log.Fatal(e)
	}
	return subcommands.ExitSuccess
}

type Game struct {
	Id     string
	White  string
	Black  string
	Time   time.Time
	Date   string
	Site   string
	Size   int
	Moves  []tak.Move
	Result string
}

func logGames(repo *logs.Repository, client *playtak.Commands, server string, out string) error {
	e := os.MkdirAll(out, 0755)
	if e != nil {
		return e
	}
	games := make(map[string]*Game)
	for line := range client.Recv() {
		if strings.HasPrefix(line, "GameList Add") {
			if g := addGame(server, games, line); g != nil {
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
				render(repo, g, out)
				delete(games, no)
			}
		}
	}
	return nil
}

func addGame(server string, games map[string]*Game, line string) *Game {
	words := strings.Split(line, " ")
	no := strings.SplitN(words[2], "#", 2)[1]
	g := &Game{
		Time:  time.Now(),
		Id:    no,
		White: words[3],
		Black: strings.TrimRight(words[5], ","),
		Size:  int(words[6][0] - '0'),
		Date:  time.Now().Format("2006-01-02"),
		Site:  server,
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
		return true
	}
	return false
}

func render(repo *logs.Repository, g *Game, dir string) {
	p := ptn.PTN{}
	p.Tags = []ptn.Tag{
		{Name: "Size", Value: strconv.Itoa(g.Size)},
		{Name: "Date", Value: g.Date},
		{Name: "Time", Value: g.Time.UTC().Format(time.RFC3339)},
		{Name: "Player1", Value: g.White},
		{Name: "Player2", Value: g.Black},
		{Name: "Result", Value: g.Result},
		{Name: "Id", Value: g.Id},
	}
	for i, m := range g.Moves {
		if i%2 == 0 {
			p.Ops = append(p.Ops, &ptn.MoveNumber{Number: i/2 + 1})
		}
		p.Ops = append(p.Ops, &ptn.Move{Move: m})
	}
	p.Ops = append(p.Ops, &ptn.Result{Result: g.Result})
	if repo != nil {
		g := PTNGame(&p)
		if g != nil {
			err := repo.InsertGame(g)
			if err != nil {
				log.Printf("insert %d: %v", g.ID, err)
			}
		}
	}
	out := p.Render()
	dir = path.Join(dir, g.Date)
	if e := os.MkdirAll(dir, 0755); e != nil {
		log.Printf("mkdir: %v", e)
		return
	}
	e := ioutil.WriteFile(fmt.Sprintf("%s/%s.ptn", dir, g.Id),
		[]byte(out), 0644)
	if e != nil {
		log.Printf("write game: %v", e)
	}
}
