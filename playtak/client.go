package playtak

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Client struct {
	conn net.Conn

	Debug bool

	Recv     chan string
	send     chan string
	shutdown chan struct{}
	wg       sync.WaitGroup
}

func (c *Client) Connect(host string) error {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}
	c.conn = conn
	c.Recv = make(chan string)
	c.send = make(chan string)
	c.shutdown = make(chan struct{})
	c.wg.Add(2)
	go c.recvThread()
	go c.sendThread()
	return nil
}

func (c *Client) recvThread() {
	r := bufio.NewReader(c.conn)
	defer c.wg.Done()
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			close(c.Recv)
			panic(err)
		}
		// trim the newline
		line = line[:len(line)-1]
		if c.Debug {
			log.Printf("< %s", line)
		}
		select {
		case c.Recv <- line:
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
			fmt.Fprintf(c.conn, "%s\n", line)
		case <-c.shutdown:
			return
		}
	}
}

func (c *Client) SendCommand(words ...string) {
	c.send <- strings.Join(words, " ")
}

func (c *Client) SendClient(name string) {
	c.SendCommand("Client", name)
}

func (c *Client) Login(user, pass string) error {
	for line := range c.Recv {
		if strings.HasPrefix(line, "Login ") {
			break
		}
	}
	if pass == "" {
		c.SendCommand("Login", user)
	} else {
		c.SendCommand("Login", user, pass)
	}
	for line := range c.Recv {
		if line == "Login or Register" {
			return errors.New("bad password")
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
