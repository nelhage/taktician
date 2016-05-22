package main

import (
	"time"

	"github.com/nelhage/taktician/tak"
)

type Game struct {
	id       string
	opponent string
	color    tak.Color
	size     int
	time     time.Duration
}
