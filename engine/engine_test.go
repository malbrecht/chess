package engine

import (
	"github.com/malbrecht/chess"
	"log"
)

func ExampleEngine() {
	var e Engine
	defer e.Quit()

	e.SetPosition(chess.MustParseFen(""))
	for info := range e.SearchDepth(6) {
		if err := info.Err(); err != nil {
			log.Fatal(err)
		} else if move, ok := info.BestMove(); ok {
			log.Print("the best move is", move)
		} else {
			log.Print(info.Pv(), info.Stats())
		}
	}
}
