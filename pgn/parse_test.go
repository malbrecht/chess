package pgn

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

type parseTest struct {
	name   string
	input  string
	games  []tgame
	errors []string
}

type ttags map[string]string

type tnode struct {
	move      string
	comment   string
	nags      []int
	variation []tnode
}

type tgame struct {
	tags  ttags
	nodes []tnode
}

var parseTests = []parseTest{
	{"basic",
		`[Result "*"] 1. e4 e5 2. Nf3 *`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5"},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"comment",
		`[Result "*"] 1. e4 { comment } e5 2. Nf3 {c1} {c2} *`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4", comment: "comment"},
			{move: "e5"},
			{move: "Nf3", comment: "c1 c2"},
		}}},
		nil,
	},
	{"root node comment",
		`[Result "*"] { comment } 1. e4 e5 2. Nf3 *`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--", comment: "comment"},
			{move: "e4"},
			{move: "e5"},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"annotations",
		`[Result "*"] 1. e4? e5!? $3 2. Nf3 $45 $45 $46 $3 *`, // $45 duplicate

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4", nags: []int{2}},
			{move: "e5", nags: []int{3, 5}},
			{move: "Nf3", nags: []int{3, 45, 46}},
		}}},
		nil,
	},
	{"missing result tag",
		`[White "John"] 1. e4 e5 2. Nf3 *`,

		[]tgame{{ttags{
			"White":  "John",
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5"},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"missing game result",
		`[Result "*"] 1. e4 e5 2. Nf3`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5"},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"with FEN tag", `
		[FEN "8/8/8/8/1K6/2p1R3/2k5/4R3 b - - 0 1"]
		[Result "1-0"]

		52...Kb2 53.R1e2+ 1-0`,

		[]tgame{{ttags{
			"FEN":    "8/8/8/8/1K6/2p1R3/2k5/4R3 b - - 0 1",
			"Result": "1-0",
		}, []tnode{
			{move: "--"},
			{move: "Kb2"},
			{move: "R1e2+"},
		}}},
		nil,
	},
	{"multiple games",
		`[Result "*"] 1. e4 e5 2. Nf3 *
		[Result "0-1"] 1. d4 d5 2. c4 0-1`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5"},
			{move: "Nf3"},
		}}, {ttags{
			"Result": "0-1",
		}, []tnode{
			{move: "--"},
			{move: "d4"},
			{move: "d5"},
			{move: "c4"},
		}}},
		nil,
	},
	{"variation",
		`[Result "*"] 1. e4 e5 (1... d5) 2. Nf3 *`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5", variation: []tnode{{move: "--"}, {move: "d5"}}},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"multiple variations (nested)",
		`[Result "*"] 1. e4 e5 (d5 (Nf6)) 2. Nf3 *`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5", variation: []tnode{
				{move: "--"}, {move: "d5", variation: []tnode{
					{move: "--"}, {move: "Nf6"},
				}},
			}},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"multiple variations (unnested)",
		`[Result "*"] 1. e4 e5 (d5) (Nf6) 2. Nf3 *`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5", variation: []tnode{
				{move: "--"}, {move: "d5", variation: []tnode{
					{move: "--"}, {move: "Nf6"},
				}},
			}},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"empty variation",
		`[Result "*"] 1. e4 e5 () (1... d5) 2. Nf3 *`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5", variation: []tnode{{move: "--"}, {move: "d5"}}},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"commented variation",
		`[Result "*"] 1. e4 e5 ({also possible} d5 {scandinavian}) 2. Nf3 *`,

		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5", variation: []tnode{
				{move: "--", comment: "also possible"},
				{move: "d5", comment: "scandinavian"},
			}},
			{move: "Nf3"},
		}}},
		nil,
	},
	{"unescape string",
		`[Event "a\"b"] [Result "*"] 1. e4 e5 2. Nf3 *`,

		[]tgame{{ttags{
			"Event":  `a"b`,
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5"},
			{move: "Nf3"},
		}}},
		nil,
	},

	// errors

	{"parse error",
		`[White "John" 1. e4 e5 2. Nf3 *`,

		nil,
		[]string{`1:14: expected ']', got <movenr>`},
	},
	{"lex error",
		`[Result "*"] 1. e4 e5 & 2. Nf3 *`,

		nil,
		[]string{`1:23: unexpected character: U+0026 '&'`},
	},
	{"recovering",
		`[White "John" 
		[Result "*"]
		
		1. d4 d5 2. c4 *

		[Result "*"] 
		
		1. e4 e5 2. Nf3 *
		`,
		[]tgame{{ttags{
			"Result": "*",
		}, []tnode{
			{move: "--"},
			{move: "e4"},
			{move: "e5"},
			{move: "Nf3"},
		}}},
		[]string{
			`1:14: expected ']', got '['`,
			`4:1: no game tags found`,
		},
	},
	{"game result mismatch",
		`[Result "1-0"] 1. e4 e5 2. Nf3 1/2-1/2`,

		nil,
		[]string{`1:31: game result "1/2-1/2" differs from Result tag "1-0"`},
	},
}

func collectGames(t *parseTest) (games []tgame, errors []string) {
	var db DB
	errs := db.Parse(t.input)
	for _, e := range errs {
		errors = append(errors, e.Error())
	}
	for _, g := range db.Games {
		if err := db.ParseMoves(g); err != nil {
			errors = append(errors, err.Error())
			continue
		}
		games = append(games, tgame{
			tags:  g.Tags,
			nodes: collectVariation(g.Root),
		})
	}
	return games, errors
}

func collectVariation(root *Node) []tnode {
	if root == nil {
		return nil
	}
	var nodes []tnode
	for nd := root; nd != nil; nd = nd.Next {
		move := "--"
		if nd.Parent != nil {
			move = nd.Move.San(nd.Parent.Board)
		}
		var nags []int
		for _, n := range nd.Nags {
			nags = append(nags, int(n))
		}
		sort.Ints(nags)

		nodes = append(nodes, tnode{
			move:      move,
			nags:      nags,
			comment:   strings.Join(nd.Comment, " "),
			variation: collectVariation(nd.Variation),
		})
	}
	return nodes
}

func TestParse(t *testing.T) {
	for _, test := range parseTests {
		games, errors := collectGames(&test)
		if !reflect.DeepEqual(games, test.games) {
			t.Errorf("%q: incorrect game tree\n", test.name)
			t.Errorf("got:  %v\n", games)
			t.Errorf("want: %v\n", test.games)
		}
		if !reflect.DeepEqual(errors, test.errors) {
			t.Errorf("%q: incorrect error set\n", test.name)
			t.Errorf("got:  %v\n", errors)
			t.Errorf("want: %v\n", test.errors)
		}
	}
}
