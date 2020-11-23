package tei

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Client struct {
	cmd *exec.Cmd

	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser

	read  *bufio.Reader
	write io.Writer

	gameid int
}

func NewClient(cmdline []string) (*Client, error) {
	cmd := &exec.Cmd{
		Args: cmdline,
	}
	if path, err := exec.LookPath(cmdline[0]); err != nil {
		return nil, err
	} else {
		cmd.Path = path
	}

	cl := &Client{
		cmd: cmd,
	}

	if stdin, err := cmd.StdinPipe(); err != nil {
		cl.Close()
		return nil, err
	} else {
		cl.stdinPipe = stdin
		cl.write = stdin
	}

	if stdout, err := cmd.StdoutPipe(); err != nil {
		cl.Close()
		return nil, err
	} else {
		cl.stdoutPipe = stdout
		cl.read = bufio.NewReader(stdout)
	}

	err := cl.cmd.Start()
	if err != nil {
		cl.Close()
		return nil, err
	}

	if _, err := cl.sendCommand("tei", "teiok"); err != nil {
		cl.Close()
		return nil, err
	}

	return cl, nil
}

func (c *Client) NewGame(size int) (ai.TakPlayer, error) {
	c.gameid += 1
	if _, err := c.sendCommand(fmt.Sprintf("teinewgame %d", size), ""); err != nil {
		return nil, err
	}
	return &player{
		client: c,
		gameid: c.gameid,
	}, nil
}

func (c *Client) Close() {
	if c.write != nil {
		c.sendCommand("quit", "")
	}
	if c.stdinPipe != nil {
		c.stdinPipe.Close()
	}
	if c.stdoutPipe != nil {
		c.stdoutPipe.Close()
	}
	c.cmd.Wait()
}

func (c *Client) sendCommand(cmd string, expect string) ([]string, error) {
	if _, err := fmt.Fprintln(c.write, cmd); err != nil {
		return nil, err
	}
	if expect == "" {
		return nil, nil
	}

	for {
		line, err := c.read.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = line[:len(line)-1]
		words := strings.Split(line, " ")
		if words[0] == expect {
			return words, nil
		}
	}
}

type player struct {
	client *Client
	gameid int
}

func (p *player) GetMove(ctx context.Context, pos *tak.Position) tak.Move {
	if p.gameid != p.client.gameid {
		panic("bad gameid: calling GetMove on a dead player")
	}
	tps := ptn.FormatTPS(pos)
	_, err := p.client.sendCommand(fmt.Sprintf("position tps %s", tps), "")
	if err != nil {
		panic(fmt.Sprintf("send position: %v", err))
	}
	goCmd := "go"
	if deadline, ok := ctx.Deadline(); ok {
		timeoutMS := deadline.Sub(time.Now()) / time.Millisecond
		goCmd = fmt.Sprintf("%s movetime %d", goCmd, timeoutMS)
	}
	bestmove, err := p.client.sendCommand(goCmd, "bestmove")
	if len(bestmove) != 2 {
		panic("bad bestmove")
	}
	mv, err := ptn.ParseMove(bestmove[1])
	if err != nil {
		panic(fmt.Sprintf("unable to parse move: %q", bestmove[1]))
	}
	return mv
}
