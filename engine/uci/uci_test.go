package uci

import (
	"bufio"
	"fmt"
	"github.com/malbrecht/chess"
	"github.com/malbrecht/chess/engine"
	"io"
	"log"
	"os"
	"testing"
	"text/tabwriter"
	"time"
)

var stdout = os.Stdout

func init() {
	CommunicationTimeout = 1 * time.Second
}

func TestStockfish(t *testing.T) {
	if true {
		return
	}

	var logger *log.Logger //= log.New(stdout, "", log.LstdFlags)

	e, err := Run("stockfish", nil, logger)
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer e.Quit()

	opt := e.Options()
	w := tabwriter.NewWriter(stdout, 1, 8, 0, ' ', 0)
	for k, v := range opt {
		fmt.Fprintln(w, k, "\t", v)
	}
	w.Flush()

	board := chess.MustParseFen("5r2/pqN1Qpk1/2r3pp/2n1R3/5R2/6P1/4PPKP/8 w - - 0 1")
	e.SetPosition(board)
	for info := range e.SearchDepth(18) {
		if info.Err() != nil {
			t.Fatalf("%s", info.Err())
		}
		if m, ok := info.BestMove(); ok {
			log.Println("bestmove:", m.San(board))
		} else if pv := info.Pv(); pv != nil {
			log.Println("pv:", pv)
			log.Println("stats:", info.Stats())
		} else {
			log.Println("stats:", info.Stats(), info.(Info).line)
		}
	}
}

type optionTest struct {
	name  string
	typ   string
	other string
	set   string
	value interface{}
}

var optionTests = []optionTest{
	{"number option 1", "spin", "default 5 min 1 max 10", "", 5},
	{"number option 2", "spin", "default 5 min 1 max 10", "7", 7},
	{"string option 1", "string", "default Ab Cd", "", "Ab Cd"},
	{"string option 2", "string", "default Ab Cd", "xyz", "xyz"},
	{"bool option 1", "check", "", "", false},
	{"bool option 2", "check", "", "true", true},
}

type infoTest struct {
	line     string
	bestmove *chess.Move
	pvscore  int
	stats    *engine.Stats
}

var infoTests = []infoTest{
	{"info nodes 1000 time 6789", nil, 0, &engine.Stats{0, 0, 1000, 6789 * time.Millisecond}},
	{"info pv e7e5 g1f3 b8c3 f1b5 score cp 29", nil, -29, nil},
	{"bestmove e7e5 ponder g1f3", &chess.Move{chess.E7, chess.E5, 0}, 0, nil},
}

func fakeEngine(r io.Reader, w io.WriteCloser) {
	buf := bufio.NewReader(r)
	for {
		line, _, err := buf.ReadLine()
		if err != nil {
			return
		}
		switch field := tokenise(string(line)); field.next() {
		case "uci":
			for _, o := range optionTests {
				fmt.Fprintf(w, "option name %s type %s %s\n", o.name, o.typ, o.other)
			}
			fmt.Fprintln(w, "uciok")
		case "isready":
			fmt.Fprintln(w, "readyok")
		case "setoption":
			// ignore
		case "go":
			for _, i := range infoTests {
				fmt.Fprintln(w, i.line)
			}
		case "quit":
			w.Close()
			return
		}
	}
}

func TestEngine(t *testing.T) {
	var logger *log.Logger //= log.New(stdout, "", log.LstdFlags)

	r0, w0 := io.Pipe()
	r1, w1 := io.Pipe()
	go fakeEngine(r1, w0)
	e, err := initialise(r0, w1, w1, logger)
	if err != nil {
		log.Fatal("engine initialisation failed:", err)
	}
	defer e.Quit()

	// test options
	opts := e.Options()
	if opts == nil {
		t.Fatal("no options returned")
	}
	for _, o := range optionTests {
		opt := opts[o.name]
		if opt == nil {
			t.Errorf("option %q not found", o.name)
			continue
		}
		if o.set != "" {
			opt.Set(o.set)
		}
		switch want := o.value.(type) {
		case string:
			s := opt.(*StringOption)
			if got := s.String(); got != want {
				t.Errorf("option %q: want %q, got %q", o.name, want, got)
			}
		case int:
			i := opt.(*IntOption)
			if got := i.Int(); got != want {
				t.Errorf("option %q: want %d, got %d", o.name, want, got)
			}
		case bool:
			b := opt.(*BoolOption)
			if got := b.Bool(); got != want {
				t.Errorf("option %q: want %v, got %v", o.name, want, got)
			}
		}
	}

	// test search
	board := chess.MustParseFen("")
	board = board.MakeMove(chess.Move{chess.E2, chess.E4, 0})
	e.SetPosition(board)

	infoc := e.SearchDepth(1)
	for _, i := range infoTests {
		info := <-infoc
		if info == nil {
			t.Fatal("got nil info instead of:", i.line)
		}
		if err := info.Err(); err != nil {
			t.Fatal("search returned error:", err)
		}
		if move, ok := info.BestMove(); ok {
			if move != *i.bestmove {
				t.Errorf("bestmove mismatch: got %v, want %v: %s",
					move, *i.bestmove, info.(Info).line)
			}
		} else if pv := info.Pv(); pv != nil {
			if pv.Score != i.pvscore {
				t.Errorf("score mismatch: got %d, want %d: %s",
					pv.Score, i.pvscore, info.(Info).line)
			}
		} else {
			stats := info.Stats()
			if *stats != *i.stats {
				t.Errorf("stats mismatch: got %v, want %v: %s",
					stats, i.stats, info.(Info).line)
			}
		}
	}
	if info := <-infoc; info != nil {
		t.Error("spurious info:", info.(Info))
	}
}
