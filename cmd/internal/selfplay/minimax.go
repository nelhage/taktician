package selfplay

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"reflect"

	"github.com/nelhage/taktician/ai"
)

func buildFactory(cfg *Config, player string, conf string, ws string) AIFactory {
	if player == "minimax" {
		return buildMinimaxFactory(cfg, player, conf, ws)
	} else if player == "mcts" {
		return buildMCTSFactory(cfg, player, conf, ws)
	} else {
		panic(fmt.Sprintf("unknown engine: %s", player))
	}
}

type MinimaxFactory struct {
	cfg ai.MinimaxConfig
}

func (m *MinimaxFactory) GetPlayer() ai.TakPlayer {
	return ai.NewMinimax(m.cfg)
}

func (m *MinimaxFactory) String() string {
	return fmt.Sprintf("minimax@%d", m.cfg.Depth)
}

func buildMinimaxFactory(cfg *Config, player string, conf string, ws string) AIFactory {
	weights := ai.DefaultWeights[cfg.Size]
	if cfg.Zero {
		weights = ai.Weights{}
	}
	if ws != "" {
		if err := json.Unmarshal([]byte(ws), &weights); err != nil {
			log.Fatal("weights:", err)
		}
	}
	mmcfg := ai.MinimaxConfig{
		Depth: cfg.Depth,
		Size:  cfg.Size,
		Debug: cfg.Debug,
	}
	if conf != "" {
		if err := json.Unmarshal([]byte(conf), &mmcfg); err != nil {
			log.Fatal("conf:", err)
		}
	}
	return &MinimaxFactory{mmcfg}
}

func perturbWeights(p float64, w ai.Weights) ai.Weights {
	r := reflect.Indirect(reflect.ValueOf(&w))
	typ := r.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() != reflect.Int {
			continue
		}
		v := r.Field(i).Interface().(int)
		adj := rand.NormFloat64() * p
		v = int(float64(v) * (1 + adj))
		r.Field(i).SetInt(int64(v))
	}

	return w
}
