package gencorpus

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/google/subcommands"
	"github.com/jmoiron/sqlx"
	"github.com/nelhage/taktician/ptn"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"
)

type Command struct {
	minRating int
	output    string
}

func (*Command) Name() string     { return "gencorpus" }
func (*Command) Synopsis() string { return "Generate a corpus of playtak positions" }
func (*Command) Usage() string {
	return `gencorpus [flags] GAMES.db
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.IntVar(&c.minRating, "min-rating", 1600, "minimum rating to consider")
	flags.StringVar(&c.output, "output", "data/corpus.parquet", "output file")
}

type GameRow struct {
	Id   int32 `db:"id" parquet:"name=id, type=INT32, encoding=DELTA_BINARY_PACKED"`
	Size int32 `db:"size" parquet:"name=size, type=INT32, encoding=PLAIN"`

	PlayerWhite string `db:"player_white" parquet:"name=player_white, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	PlayerBlack string `db:"player_black" parquet:"name=player_black, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`

	TimerTime int32 `db:"timertime" parquet:"name=timer_time, type=INT32, encoding=PLAIN"`
	TimerInc  int32 `db:"timerinc" parquet:"name=timer_inc, type=INT32, encoding=PLAIN"`

	RatingWhite int `db:"rating_white" parquet:"name=rating_white, type=INT32, encoding=DELTA_BINARY_PACKED"`
	RatingBlack int `db:"rating_black" parquet:"name=rating_black, type=INT32, encoding=DELTA_BINARY_PACKED"`

	Pieces    int `db:"pieces"`
	Capstones int `db:"capstones"`

	PTN string `db:"ptn"`
}

type Position struct {
	GameRow `parquet:"name=game"`

	Ply      int32  `parquet:"name=ply, type=INT32, convertedtype=UINT_32, encoding=DELTA_BINARY_PACKED"`
	Position string `parquet:"name=position, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	Move     string `parquet:"name=move, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	sql, err := sqlx.Open("sqlite3", flag.Arg(0))
	if err != nil {
		log.Fatalf("open %s: %v", flag.Arg(0), err)
	}

	rows, e := sql.Queryx(`
SELECT g.id, g.size,
       g.player_white, g.player_black,
       g.timertime, g.timerinc,
       g.rating_white, g.rating_black,
       g.pieces, g.capstones,
       p.ptn
FROM games g, ptns p
WHERE g.rating_white >= ?
  AND g.rating_black >= ?
  AND p.id = g.id
  AND p.id IS NOT NULL
`, c.minRating, c.minRating)
	if e != nil {
		log.Fatal("select: ", e)
	}
	defer rows.Close()

	w, err := os.Create(c.output)
	if err != nil {
		log.Fatalf("create %q: %v", c.output, err)
	}

	defer w.Close()

	pw, err := writer.NewParquetWriterFromWriter(w, &Position{}, 4)
	if err != nil {
		log.Fatal("Can't create parquet writer:", err)
	}

	pw.CompressionType = parquet.CompressionCodec_ZSTD

	for rows.Next() {
		var row GameRow
		if err := rows.StructScan(&row); err != nil {
			log.Fatalf("read row: %v", err)
		}

		positions, err := ptn.ParsePTN(strings.NewReader(row.PTN))
		if e != nil {
			log.Printf("parse %d: %v", row.Id, err)
			continue
		}

		it := positions.Iterator()

		if (row.Pieces != -1 && row.Pieces != it.Position().Config().Pieces) ||
			(row.Capstones != -1 && row.Capstones != it.Position().Config().Capstones) {
			continue
		}

		for it.Next() {
			pos := it.Position()
			var move string
			if mv := it.PeekMove(); mv.Type != 0 {
				move = ptn.FormatMove(mv)
			}
			out := &Position{
				GameRow:  row,
				Ply:      int32(pos.MoveNumber()),
				Position: ptn.FormatTPS(pos),
				Move:     move,
			}
			if err := pw.Write(out); err != nil {
				log.Fatalf("write parquet: %v", err)
			}
		}
		if err := it.Err(); err != nil {
			log.Fatalf("iterating %d: %v", row.Id, err)
		}
	}

	defer func() {
		if err := pw.WriteStop(); err != nil {
			log.Fatalf("writing parquet: %v", err)
		}
	}()

	return subcommands.ExitSuccess
}
