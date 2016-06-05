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

type Client struct {
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

func (c *Client) Error() error {
	return c.err
}

func (c *Client) Connect(host string) error {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}
	c.conn = conn
	c.recv = make(chan string)
	c.send = make(chan string)
	c.shutdown = make(chan struct{})
	c.wg.Add(2)
	go c.recvThread()
	go c.sendThread()
	return nil
}

func (c *Client) logSent(l string) {
	c.last.Lock()
	defer c.last.Unlock()
	c.last.buf[c.last.i] = l
	c.last.i = (c.last.i + 1) % len(c.last.buf)
}

func (c *Client) lastSent() []string {
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

func (c *Client) recvThread() {
	r := bufio.NewReader(c.conn)
	defer c.wg.Done()
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			close(c.recv)
			c.err = err
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
		}
		select {
		case c.recv <- line:
		case <-c.shutdown:
			return
		}
	}
}

func (c *Client) ping() {
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

func (c *Client) sendThread() {
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

func (c *Client) SendCommand(words ...string) {
	c.send <- strings.Join(words, " ")
}

func (c *Client) Recv() <-chan string {
	return c.recv
}

func (c *Client) SendClient(name string) {
	c.SendCommand("Client", name)
}

func (c *Client) Login(user, pass string) error {
	for line := range c.recv {
		if strings.HasPrefix(line, "Login ") {
			break
		}
	}
	if pass == "" {
		c.SendCommand("Login", user)
	} else {
		c.SendCommand("Login", user, pass)
	}
	for line := range c.recv {
		if line == "Login or Register" {
			return errors.New("bad password")
		}
		if line == "You're already logged in" {
			return errors.New("user is already logged in")
		}
		if strings.HasPrefix(line, "Welcome ") {
			return nil
		}
	}
	return errors.New("login failed")
}

func (c *Client) LoginGuest() error {
	return c.Login("Guest", "")
}

func (c *Client) Shutdown() {
	close(c.shutdown)
	c.wg.Wait()
	c.conn.Close()
}

var shoutRE = regexp.MustCompile(`^Shout <([^> ]+)> (.+)$`)

func ParseShout(line string) (string, string) {
	gs := shoutRE.FindStringSubmatch(line)
	if gs == nil {
		return "", ""
	}
	return gs[1], gs[2]
}
