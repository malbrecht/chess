package engine

import (
	"log"
)

func ExampleEngine() {
	var e Engine
	defer e.Quit()

	e.SetPosition("")
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
