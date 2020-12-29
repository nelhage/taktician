package tei

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Client struct {
	DebugPfx string

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

	cmd.Stderr = os.Stderr

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

func (c *Client) NewGame(size int) (*Player, error) {
	c.gameid += 1
	if _, err := c.sendCommand(fmt.Sprintf("teinewgame %d", size), ""); err != nil {
		return nil, err
	}
	return &Player{
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
	if c.DebugPfx != "" {
		log.Printf("[%s]> %s", c.DebugPfx, cmd)
	}
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
		if c.DebugPfx != "" {
			log.Printf("[%s]< %s", c.DebugPfx, line)
		}
		line = strings.TrimSpace(line)
		words := strings.Fields(line)
		if words[0] == expect {
			return words, nil
		}
	}
}

type Player struct {
	client *Client
	gameid int
}

func (p *Player) TEIGetMove(ctx context.Context, pos *tak.Position, tc *TimeControl) (tak.Move, error) {
	if p.gameid != p.client.gameid {
		panic("bad gameid: calling GetMove on a dead player")
	}
	tps := ptn.FormatTPS(pos)
	_, err := p.client.sendCommand(fmt.Sprintf("position tps %s", tps), "")
	if err != nil {
		return tak.Move{}, fmt.Errorf("send position: %w", err)
	}
	goCmd := []string{"go"}
	if deadline, ok := ctx.Deadline(); ok {
		goCmd = append(goCmd, "movetime", formatTime(deadline.Sub(time.Now())))
	}
	if tc != nil {
		times := []struct {
			key string
			dur time.Duration
		}{
			{"wtime", tc.White},
			{"btime", tc.Black},
			{"winc", tc.WInc},
			{"binc", tc.BInc},
		}
		for _, t := range times {
			if t.dur != 0 {
				if t.dur < time.Millisecond {
					return tak.Move{}, errors.New("Timeout too short")
				}
				goCmd = append(goCmd, t.key, formatTime(t.dur))
			}
		}
	}
	bestmove, err := p.client.sendCommand(strings.Join(goCmd, " "), "bestmove")
	if err != nil {
		return tak.Move{}, fmt.Errorf("tei: server error: %w", err)
	}
	if len(bestmove) != 2 {
		return tak.Move{}, fmt.Errorf("bad bestmove: %v", bestmove)
	}
	mv, err := ptn.ParseMove(bestmove[1])
	if err != nil {
		return tak.Move{}, fmt.Errorf("tei: unparseable move: %q", bestmove[1])
	}
	return mv, nil
}

func (p *Player) GetMove(ctx context.Context, pos *tak.Position) tak.Move {
	mv, err := p.TEIGetMove(ctx, pos, nil)
	if err != nil {
		panic(fmt.Sprintf("TEI: GetMove: %s", err.Error()))
	}
	return mv
}
