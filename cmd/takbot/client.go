package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"

	"nelhage.com/tak/ai"
	"nelhage.com/tak/tak"
)

type client struct {
	ai   ai.TakPlayer
	conn net.Conn
	p    *tak.Position

	debug bool

	recv     chan string
	send     chan string
	shutdown chan struct{}
}

func (c *client) Connect(host string) error {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}
	c.conn = conn
	c.recv = make(chan string)
	c.send = make(chan string)
	c.shutdown = make(chan struct{})
	go c.recvThread()
	go c.sendThread()
	return nil
}

func (c *client) recvThread() {
	r := bufio.NewReader(c.conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			close(c.recv)
			panic(err)
		}
		if c.debug {
			log.Printf("< %s", line)
		}
		select {
		case c.recv <- line:
		case <-c.shutdown:
			return
		}
	}
}

func (c *client) sendThread() {
	for {
		select {
		case line := <-c.send:
			if c.debug {
				log.Printf("> %s", line)
			}
			fmt.Fprintf(c.conn, "%s\n", line)
		case <-c.shutdown:
			return
		}
	}
}

func (c *client) sendCommand(words ...string) {
	c.send <- strings.Join(words, " ")
}

func (c *client) SendClient(name string) {
	c.sendCommand("Client", name)
}

func (c *client) Login(user, pass string) error {
	for line := range c.recv {
		if strings.HasPrefix(line, "Login ") {
			break
		}
	}
	if pass == "" {
		c.sendCommand("Login", user)
	} else {
		c.sendCommand("Login", user, pass)
	}
	for line := range c.recv {
		if line == "Login or Register" {
			return errors.New("bad password")
		}
		if strings.HasPrefix(line, "Welcome ") {
			return nil
		}
	}
	return errors.New("login failed")
}

func (c *client) LoginGuest() error {
	return c.Login("Guest", "")
}
