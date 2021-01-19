package playtak

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Client interface {
	Recv() <-chan string
	SendCommand(...string)

	Error() error
	Shutdown()
}

type client struct {
	conn net.Conn

	Debug bool

	err error

	recv     chan string
	send     chan string
	shutdown chan struct{}
	wg       sync.WaitGroup

	last struct {
		sync.Mutex
		buf [5]string
		i   int
	}
}

func (c *client) Error() error {
	return c.err
}

func Dial(debug bool, host string) (Client, error) {
	client := &client{
		Debug: debug,
	}
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	client.conn = conn
	client.recv = make(chan string)
	client.send = make(chan string)
	client.shutdown = make(chan struct{})
	client.wg.Add(2)
	go client.recvThread()
	go client.sendThread()
	return client, nil
}

func (c *client) logSent(l string) {
	c.last.Lock()
	defer c.last.Unlock()
	c.last.buf[c.last.i] = l
	c.last.i = (c.last.i + 1) % len(c.last.buf)
}

func (c *client) lastSent() []string {
	out := make([]string, 0, len(c.last.buf))
	c.last.Lock()
	defer c.last.Unlock()
	for i := 1; i < len(c.last.buf); i++ {
		j := (c.last.i - i + len(c.last.buf)) % len(c.last.buf)
		if c.last.buf[j] != "" {
			out = append(out, c.last.buf[j])
		}
	}
	return out
}

func (c *client) recvThread() {
	r := bufio.NewReader(c.conn)
	defer c.wg.Done()
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			c.err = err
			close(c.recv)
			c.conn.Close()
			return
		}
		// trim the newline
		line = line[:len(line)-1]
		if c.Debug {
			log.Printf("< %s", line)
		}
		if line == "NOK" {
			log.Printf("NOK! last messages:")
			for _, m := range c.lastSent() {
				log.Printf(" - `%s`", m)
			}
			c.err = errors.New("server sent NOK")
			close(c.recv)
			c.conn.Close()
			return
		}
		select {
		case c.recv <- line:
		case <-c.shutdown:
			return
		}
	}
}

func (c *client) ping() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.send <- "PING"
		case <-c.shutdown:
			return
		}
	}
}

func (c *client) sendThread() {
	go c.ping()
	defer c.wg.Done()
	for {
		select {
		case line := <-c.send:
			if c.Debug {
				log.Printf("> %s", line)
			}
			c.logSent(line)
			fmt.Fprintf(c.conn, "%s\n", line)
		case <-c.shutdown:
			return
		}
	}
}

func (c *client) SendCommand(words ...string) {
	c.send <- strings.Join(words, " ")
}

func (c *client) Recv() <-chan string {
	return c.recv
}

func (c *client) Shutdown() {
	close(c.shutdown)
	c.wg.Wait()
	c.conn.Close()
}

var (
	tellRE      = regexp.MustCompile(`^Tell <([^> ]+)> (.+)$`)
	shoutRE     = regexp.MustCompile(`^Shout <([^> ]+)> (.+)$`)
	shoutRoomRE = regexp.MustCompile(`^ShoutRoom (\S+) <([^> ]+)> (.+)$`)
)

func ParseTell(line string) (string, string) {
	gs := tellRE.FindStringSubmatch(line)
	if gs == nil {
		return "", ""
	}
	return gs[1], gs[2]
}

func ParseShout(line string) (string, string) {
	gs := shoutRE.FindStringSubmatch(line)
	if gs == nil {
		return "", ""
	}
	return gs[1], gs[2]
}

func ParseShoutRoom(line string) (string, string, string) {
	gs := shoutRoomRE.FindStringSubmatch(line)
	if gs == nil {
		return "", "", ""
	}
	return gs[1], gs[2], gs[3]
}
