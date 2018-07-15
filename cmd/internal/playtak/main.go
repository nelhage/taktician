package playtak

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/playtak"
	"github.com/nelhage/taktician/playtak/bot"
)

type Command struct {
	server    string
	user      string
	pass      string
	accept    string
	observe   string
	gameTime  time.Duration
	increment time.Duration
	size      int
	once      bool
	takbot    string

	friendly bool
	fpa      string
	logFile  string

	debug           int
	depth           int
	multicut        bool
	limit           time.Duration
	sort            bool
	tableMem        int64
	useOpponentTime bool

	book bool

	debugClient bool
}

func (*Command) Name() string     { return "playtak" }
func (*Command) Synopsis() string { return "Play Tak on playtak.com using the Taktician AI" }
func (*Command) Usage() string {
	return `playtak [flags]
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.StringVar(&c.server, "server", "playtak.com:10000", "playtak.com server to connect to")
	flags.StringVar(&c.user, "user", "", "username for login")
	flags.StringVar(&c.pass, "pass", "", "password for login")
	flags.StringVar(&c.accept, "accept", "", "accept a game from specified user")
	flags.StringVar(&c.observe, "observe", "", "observe a game by a specified user")
	flags.DurationVar(&c.gameTime, "time", 20*time.Minute, "Length of game to offer")
	flags.DurationVar(&c.increment, "increment", 0, "time increment to offer")
	flags.IntVar(&c.size, "size", 5, "size of game to offer")
	flags.BoolVar(&c.once, "once", false, "play a single game and exit")
	flags.StringVar(&c.takbot, "takbot", "", "challenge TakBot AI")

	flags.BoolVar(&c.friendly, "friendly", false, "play as FriendlyBot")
	flags.StringVar(&c.fpa, "fpa", "", "select an alternate FPA rule set")
	flags.StringVar(&c.logFile, "log-file", "", "Log friendly/FPA games")

	flags.IntVar(&c.debug, "debug", 1, "debug level")
	flags.IntVar(&c.depth, "depth", 0, "minimax depth")
	flags.BoolVar(&c.multicut, "multi-cut", false, "use multi-cut")
	flags.DurationVar(&c.limit, "limit", time.Minute, "time limit per move")
	flags.BoolVar(&c.sort, "sort", true, "sort moves via history heuristic")
	flags.Int64Var(&c.tableMem, "table-mem", 0, "set table size")
	flags.BoolVar(&c.useOpponentTime, "use-opponent-time", true, "think on opponent's time")

	flags.BoolVar(&c.book, "book", true, "use built-in opening book")

	flags.BoolVar(&c.debugClient, "debug-client", false, "log debug output for playtak connection")
}

const ClientName = "Taktician AI"

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.accept != "" || c.takbot != "" || c.observe != "" {
		c.once = true
	}
	var fpaRuleset FPARule
	if c.fpa != "" {
		c.friendly = true
		switch c.fpa {
		case "true", "center":
			fpaRuleset = &CenterBlack{}
		case "doublestack":
			fpaRuleset = &DoubleStack{}
		case "cairn":
			fpaRuleset = &Cairn{}
		default:
			log.Fatalf("Unknown FPA ruleset: %s", c.fpa)
		}
	}

	sigs := make(chan os.Signal, 1)
	if c.observe == "" {
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	}

	backoff := 1 * time.Second
	var b bot.Bot
	for {
		var client *playtak.Commands
		cl, err := playtak.Dial(c.debugClient, c.server)
		if err != nil {
			goto reconnect
		}
		backoff = time.Second
		client = &playtak.Commands{"", cl}
		client.SendClient(ClientName)
		if c.user != "" {
			err = client.Login(c.user, c.pass)
		} else {
			err = client.LoginGuest()
		}
		if err != nil {
			log.Fatal("login: ", err)
		}
		log.Printf("login OK")
		if c.friendly {
			b = &Friendly{
				cmd:     c,
				logFile: c.logFile,
				client:  client,
				fpa:     fpaRuleset,
			}
		} else {
			b = &Taktician{cmd: c, client: client}
		}
		for {
			if c.accept == "" && c.observe == "" {
				client.SendCommand("Seek",
					strconv.Itoa(c.size),
					strconv.Itoa(int(c.gameTime.Seconds())),
					strconv.Itoa(int(c.increment.Seconds())))
				log.Printf("Seek OK")
				if c.takbot != "" {
					client.SendCommand("Shout", "takbot: play", c.takbot)
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
						if bits[3] == c.accept {
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
						if white == c.observe || black == c.observe {
							client.SendCommand("Observe", bits[2][len("Game#"):])
						}
					case strings.HasPrefix(line, "Observe"):
						bot.ObserveGame(client, b, line)
						break recvLoop
					}
				case <-sigs:
					return subcommands.ExitSuccess
				}
			}
			if c.once {
				return subcommands.ExitSuccess
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
			return subcommands.ExitSuccess
		}
		backoff = backoff * 2
		if backoff > time.Minute {
			backoff = time.Minute
		}
	}
}
