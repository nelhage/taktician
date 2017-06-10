package ai

import (
	"encoding/json"
	"fmt"
)

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = &Weights{}

var featureNames map[string]Feature

func init() {
	featureNames = make(map[string]Feature)
	for i := Feature(0); i < MaxFeature; i++ {
		featureNames[i.String()] = i
	}
}

func (ws *Weights) MarshalJSON() ([]byte, error) {
	h := make(map[string]int64)
	for i, v := range ws {
		if v != 0 {
			h[Feature(i).String()] = v
		}
	}
	return json.Marshal(h)
}

func (ws *Weights) UnmarshalJSON(bs []byte) error {
	h := make(map[string]int64)
	e := json.Unmarshal(bs, &h)
	if e != nil {
		return e
	}
	for k, v := range h {
		f, ok := featureNames[k]
		if !ok {
			return fmt.Errorf("Unknown feature: %q", k)
		}
		ws[f] = v
	}
	return nil
}
