package selfplay

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/ai/mcts"
)

type MCTSFactory struct {
	cfg mcts.MCTSConfig
}

func (m *MCTSFactory) GetPlayer() ai.TakPlayer {
	return mcts.NewMonteCarlo(m.cfg)
}

func (m *MCTSFactory) String() string {
	return fmt.Sprintf("mcts@%s", m.cfg.Limit)
}

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
	return &MCTSFactory{mctscfg}
}
