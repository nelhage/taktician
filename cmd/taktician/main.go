package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/playtak/bot"
)

var (
	server    = flag.String("server", "playtak.com:10000", "playtak.com server to connect to")
	user      = flag.String("user", "", "username for login")
	pass      = flag.String("pass", "", "password for login")
	accept    = flag.String("accept", "", "accept a game from specified user")
	observe   = flag.String("observe", "", "observe a game by a specified user")
	gameTime  = flag.Duration("time", 20*time.Minute, "Length of game to offer")
	increment = flag.Duration("increment", 0, "time increment to offer")
	size      = flag.Int("size", 5, "size of game to offer")
	once      = flag.Bool("once", false, "play a single game and exit")
	takbot    = flag.String("takbot", "", "challenge TakBot AI")

	friendly = flag.Bool("friendly", false, "play as FriendlyBot")
	fpa      = flag.Bool("fpa", false, "play as FPABot")
	logFile  = flag.String("log-file", "", "Log friendly/FPA games")

	debug           = flag.Int("debug", 1, "debug level")
	depth           = flag.Int("depth", 0, "minimax depth")
	multicut        = flag.Bool("multi-cut", false, "use multi-cut")
	limit           = flag.Duration("limit", time.Minute, "time limit per move")
	sort            = flag.Bool("sort", true, "sort moves via history heuristic")
	tableMem        = flag.Int64("table-mem", 0, "set table size")
	useOpponentTime = flag.Bool("use-opponent-time", true, "think on opponent's time")

	book = flag.Bool("book", true, "use built-in opening book")

	debugClient = flag.Bool("debug-client", false, "log debug output for playtak connection")
)

const ClientName = "Taktician AI"

func main() {
	flag.Parse()
	if *accept != "" || *takbot != "" || *observe != "" {
		*once = true
	}

	sigs := make(chan os.Signal, 1)
	if *observe == "" {
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	}

	backoff := 1 * time.Second
	var b bot.Bot
	for {
		var client *playtak.Commands
		cl, err := playtak.Dial(*debugClient, *server)
		if err != nil {
			goto reconnect
		}
		backoff = time.Second
		client = &playtak.Commands{cl}
		client.SendClient(ClientName)
		if *user != "" {
			err = client.Login(*user, *pass)
		} else {
			err = client.LoginGuest()
		}
		if err != nil {
			log.Fatal("login: ", err)
		}
		log.Printf("login OK")
		if *friendly || *fpa {
			fb := &Friendly{client: client}
			if *fpa {
				fb.fpa = &CenterBlack{}
			}
			b = fb
		} else {
			b = &Taktician{client: client}
		}
		for {
			if *accept == "" && *observe == "" {
				client.SendCommand("Seek",
					strconv.Itoa(*size),
					strconv.Itoa(int(gameTime.Seconds())),
					strconv.Itoa(int(increment.Seconds())))
				log.Printf("Seek OK")
				if *takbot != "" {
					client.SendCommand("Shout", "takbot: play", *takbot)
				}
			}

		recvLoop:
			for {
				select {
				case line, ok := <-client.Recv():
					if !ok {
						break recvLoop
					}
					switch {
					case strings.HasPrefix(line, "Seek new"):
						bits := strings.Split(line, " ")
						if bits[3] == *accept {
							log.Printf("accepting game %s from %s", bits[2], bits[3])
							client.SendCommand("Accept", bits[2])
						}
					case strings.HasPrefix(line, "Game Start"):
						bot.PlayGame(client, b, line)
						time.Sleep(100 * time.Millisecond)
						break recvLoop
					case strings.HasPrefix(line, "Shout"):
						who, msg := playtak.ParseShout(line)
						if who != "" {
							b.HandleChat("", who, msg)
						}
					case strings.HasPrefix(line, "GameList Add"):
						bits := strings.Split(line, " ")
						white := bits[3]
						black := strings.TrimRight(bits[5], ",")
						if white == *observe || black == *observe {
							client.SendCommand("Observe", bits[2][len("Game#"):])
						}
					case strings.HasPrefix(line, "Observe"):
						bot.ObserveGame(client, b, line)
						break recvLoop
					}
				case <-sigs:
					return
				}
			}
			if *once {
				return
			}
			if client.Error() != nil {
				log.Printf("Disconnected: %v", client.Error())
				break
			}
		}
	reconnect:
		log.Printf("sleeping %s before reconnect...", backoff)
		select {
		case <-time.After(backoff):
		case <-sigs:
			return
		}
		backoff = backoff * 2
		if backoff > time.Minute {
			backoff = time.Minute
		}
	}
}
