package main

import (
	"encoding/json"
	"log"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ai/mcts"
)

func buildMCTSFactory(cfg *Config, player string, conf string, ws string) AIFactory {
	mctscfg := mcts.MCTSConfig{
		Size:  cfg.Size,
		Debug: cfg.Debug,
		Limit: cfg.Limit,
	}
	if conf != "" {
		if err := json.Unmarshal([]byte(conf), &mctscfg); err != nil {
			log.Fatal("conf:", err)
		}
	}
	return func() ai.TakPlayer {
		return mcts.NewMonteCarlo(mctscfg)
	}
}
