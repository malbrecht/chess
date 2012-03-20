package chess

import (
	"fmt"
	"reflect"
	"testing"
)

const __ = NoPiece

// pieceArray is a helper function to be able to present a chessboard
// "normally" with black on top:
//
//	pieceArray(
//		BR, BN, BB, BQ, BK, BB, BN, BR,
//		BP, BP, BP, BP, BP, BP, BP, BP,
//		__, __, __, __, __, __, __, __,
//		__, __, __, __, __, __, __, __,
//		__, __, __, __, __, __, __, __,
//		__, __, __, __, __, __, __, __,
//		WP, WP, WP, WP, WP, WP, WP, WP,
//		WR, WN, WB, WQ, WK, WB, WN, WR,
//	),
//
func pieceArray(input ...Piece) (output [64]Piece) {
	if len(input) != 64 {
		panic(fmt.Sprint("pieceArray called with", len(input), "squares"))
	}
	i := 0
	for rank := 7; rank >= 0; rank-- {
		for file := 0; file <= 7; file++ {
			output[Square(file, rank)] = input[i]
			i++
		}
	}
	return output
}

// ParseFen

type fenTest struct {
	name   string
	fen    string
	b      *Board
	fenOut string
}

var fenTests = []fenTest{
	{"empty FEN", "", &Board{
		Piece: pieceArray(
			BR, BN, BB, BQ, BK, BB, BN, BR,
			BP, BP, BP, BP, BP, BP, BP, BP,
			__, __, __, __, __, __, __, __,
			__, __, __, __, __, __, __, __,
			__, __, __, __, __, __, __, __,
			__, __, __, __, __, __, __, __,
			WP, WP, WP, WP, WP, WP, WP, WP,
			WR, WN, WB, WQ, WK, WB, WN, WR,
		),
		SideToMove: White,
		MoveNr:     1,
		EpSquare:   NoSquare,
		CastleSq:   [4]Sq{A1, A8, H1, H8}},

		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	},
	{"full FEN", "r4rk1/2pp1ppp/8/8/5P2/8/PPPPP1PP/RNBQKBNR b KQ c3 0 12", &Board{
		Piece: pieceArray(
			BR, __, __, __, __, BR, BK, __,
			__, __, BP, BP, __, BP, BP, BP,
			__, __, __, __, __, __, __, __,
			__, __, __, __, __, __, __, __,
			__, __, __, __, __, WP, __, __,
			__, __, __, __, __, __, __, __,
			WP, WP, WP, WP, WP, __, WP, WP,
			WR, WN, WB, WQ, WK, WB, WN, WR,
		),
		SideToMove: Black,
		MoveNr:     12,
		EpSquare:   C3,
		CastleSq:   [4]Sq{A1, NoSquare, H1, NoSquare}},

		"r4rk1/2pp1ppp/8/8/5P2/8/PPPPP1PP/RNBQKBNR b KQ c3 0 12",
	},
}

func TestFEN(t *testing.T) {
	for _, test := range fenTests {
		b, err := ParseFen(test.fen)
		if err != nil {
			t.Errorf("test %s failed: %s", test.name, err)
			continue
		}
		if !reflect.DeepEqual(b, test.b) {
			t.Errorf("%s: load fen failed:\n\texp: %v\n\tgot: %v",
				test.name, *test.b, *b)
		}
		if fen := test.b.Fen(); fen != test.fenOut {
			t.Errorf("%s: store fen failed:\n\texp: %v\n\tgot: %v",
				test.name, test.fenOut, fen)
		}
	}
}

// ParseMove

type parseMoveTest struct {
	input string
	move  Move
}

var parseMoveBoard = &Board{
	Piece: pieceArray(
		BR, __, __, __, BK, __, __, BR,
		BP, __, __, __, __, __, __, __,
		__, __, BN, __, BP, __, __, __,
		__, __, __, BN, WQ, WP, __, __,
		__, __, __, __, __, BP, WP, __,
		__, __, __, __, __, __, __, __,
		__, BP, __, __, __, __, __, __,
		WR, __, __, __, __, __, WK, __,
	),
	SideToMove: Black,
	MoveNr:     1,
	Rule50:     0,
	EpSquare:   G3,
	CastleSq:   [4]Sq{NoSquare, A8, NoSquare, H8},
}

var parseMoveTests = []parseMoveTest{
	{"a7a6", Move{A7, A6, NoPiece}},   // pawn move uci
	{"a6", Move{A7, A6, NoPiece}},     // pawn move san
	{"a7a5", Move{A7, A5, NoPiece}},   // double pawn move uci
	{"a5", Move{A7, A5, NoPiece}},     // double pawn move san
	{"f4g3", Move{F4, G3, NoPiece}},   // en-passant uci
	{"fxg3", Move{F4, G3, NoPiece}},   // en-passant san
	{"fg", Move{F4, G3, NoPiece}},     // very short pawn capture
	{"b2b1q", Move{B2, B1, BQ}},       // promotion uci
	{"b2b1r", Move{B2, B1, BR}},       // promotion uci
	{"b2b1b", Move{B2, B1, BB}},       // promotion uci
	{"b2b1n", Move{B2, B1, BN}},       // promotion uci
	{"b1=Q", Move{B2, B1, BQ}},        // promotion san
	{"b1/Q", Move{B2, B1, BQ}},        // promotion san
	{"b1(Q)+?", Move{B2, B1, BQ}},     // promotion san
	{"Nd4", Move{C6, D4, NoPiece}},    // knight move
	{"Nc6-d4", Move{C6, D4, NoPiece}}, // knight move long notation
	{"0-0", Move{E8, H8, NoPiece}},    // castling san
	{"O-O", Move{E8, H8, NoPiece}},    // castling pgn
	{"O-O-O", Move{E8, A8, NoPiece}},  // castling queenside
	{"e8g8", Move{E8, H8, NoPiece}},   // castling uci
	{"e8h8", Move{E8, H8, NoPiece}},   // castling uci960
	// invalid moves
	{"Nb4", Move{}},  // ambiguous move
	{"exf5", Move{}}, // the pawn is pinned
}

func TestParseMove(t *testing.T) {
	for _, test := range parseMoveTests {
		m, err := parseMoveBoard.ParseMove(test.input)
		if err != nil {
			m = Move{}
		}
		if !reflect.DeepEqual(m, test.move) {
			t.Errorf("move %s:\n\texp: %v\n\tgot: %v\n",
				test.input, test.move, m)
		}
	}
}
