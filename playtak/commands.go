package playtak

import (
	"errors"
	"strings"
)

type Commands struct {
	Client
}

func (c *Commands) SendClient(name string) {
	c.SendCommand("client", name)
}

func (c *Commands) Login(user, pass string) error {
	for line := range c.Recv() {
		if strings.HasPrefix(line, "Login ") {
			break
		}
	}
	if pass == "" {
		c.SendCommand("Login", user)
	} else {
		c.SendCommand("Login", user, pass)
	}
	for line := range c.Recv() {
		if line == "Authentication failure" {
			return errors.New("bad username or password")
		}
		if strings.HasPrefix(line, "Welcome ") {
			return nil
		}
	}
	return errors.New("login failed")
}

func (c *Commands) LoginGuest() error {
	return c.Login("Guest", "")
}

func (c *Commands) Shout(room, msg string) {
	if room == "" {
		c.SendCommand("Shout", msg)
	} else {
		c.SendCommand("ShoutRoom", room, msg)
	}
}

func (c *Commands) Tell(who, msg string) {
	c.SendCommand("Tell", who, msg)
}
