package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"reflect"

	"github.com/nelhage/taktician/ai"
)

type field struct {
	name string
	val  *int
}

func getFields(w *ai.Weights) []field {
	var fs []field

	r := reflect.Indirect(reflect.ValueOf(w))
	typ := r.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Int {
			p := r.Field(i).Addr().Interface().(*int)
			fs = append(fs, field{f.Name, p})
			continue
		}
		if f.Type.Kind() == reflect.Array &&
			f.Type.Elem().Kind() == reflect.Int {
			len := f.Type.Len()
			for j := 0; j < len; j++ {
				p := r.Field(i).Index(j).Addr().Interface().(*int)
				name := fmt.Sprintf("%s[%d]", f.Name, j)
				fs = append(fs, field{name, p})
			}
		}
	}

	return fs
}

const Stride = 100

func doSearch(cfg ai.MinimaxConfig, w ai.Weights) {
	fields := getFields(&w)
	r := rand.New(rand.NewSource(*seed))
	for {
		prev := w
		field := fields[r.Intn(len(fields))]
		oldv := *field.val
		*field.val += int(r.NormFloat64() * Stride)
		log.Printf("testing field=%v old=%d new=%d", field.name, oldv, *field.val)

		st := Simulate(&Config{
			Cfg1: cfg, Cfg2: cfg,
			W1: prev, W2: w,

			Seed: r.Int63(),

			Swap:    true,
			Games:   *games,
			Threads: *threads,
			Cutoff:  *cutoff,
			Limit:   *limit,
		})

		log.Printf("done ties=%d p1.wins=%d (%d road/%d flat) p2.wins=%d (%d road/%d flat)",
			st.Ties,
			st.Players[0].Wins, st.Players[0].RoadWins, st.Players[0].FlatWins,
			st.Players[1].Wins, st.Players[1].RoadWins, st.Players[1].FlatWins)

		if st.Players[1].Wins < st.Players[0].Wins {
			log.Printf("result=reject")
			w = prev
		} else {
			log.Printf("result=accept")
		}
		j, _ := json.Marshal(w)
		log.Printf("w=%s", j)
	}
}
