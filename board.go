// Package chess provides functionality to handle chess and chess960 board positions.
package chess

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	White = iota
	Black
)

// Piece types
const (
	NoPiece = iota << 1
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

// Pieces
const (
	WP = White | Pawn
	WN = White | Knight
	WB = White | Bishop
	WR = White | Rook
	WQ = White | Queen
	WK = White | King
	BP = Black | Pawn
	BN = Black | Knight
	BB = Black | Bishop
	BR = Black | Rook
	BQ = Black | Queen
	BK = Black | King
)

type Piece uint8

func (p Piece) Color() int { return int(p) & 0x01 }
func (p Piece) Type() int  { return int(p) &^ 0x01 }

var PieceLetters = []rune{
	'.', ',',
	'P', 'p',
	'N', 'n',
	'B', 'b',
	'R', 'r',
	'Q', 'q',
	'K', 'k',
}

var Figurines = []rune{
	'.', ',',
	0x2659, 0x265F,
	0x2658, 0x265E,
	0x2657, 0x265D,
	0x2656, 0x265C,
	0x2655, 0x265B,
	0x2654, 0x265A,
}

func pieceFromChar(c rune) Piece {
	for i := WP; i < len(PieceLetters); i++ {
		if PieceLetters[i] == c {
			return Piece(i)
		}
	}
	return NoPiece
}

// Squares

const (
	A1, B1, C1, D1, E1, F1, G1, H1 Sq = 8*iota + 0, 8*iota + 1, 8*iota + 2,
		8*iota + 3, 8*iota + 4, 8*iota + 5, 8*iota + 6, 8*iota + 7
	A2, B2, C2, D2, E2, F2, G2, H2
	A3, B3, C3, D3, E3, F3, G3, H3
	A4, B4, C4, D4, E4, F4, G4, H4
	A5, B5, C5, D5, E5, F5, G5, H5
	A6, B6, C6, D6, E6, F6, G6, H6
	A7, B7, C7, D7, E7, F7, G7, H7
	A8, B8, C8, D8, E8, F8, G8, H8
	NoSquare Sq = -1
)

var squareNames = []string{
	"a1", "b1", "c1", "d1", "e1", "f1", "g1", "h1",
	"a2", "b2", "c2", "d2", "e2", "f2", "g2", "h2",
	"a3", "b3", "c3", "d3", "e3", "f3", "g3", "h3",
	"a4", "b4", "c4", "d4", "e4", "f4", "g4", "h4",
	"a5", "b5", "c5", "d5", "e5", "f5", "g5", "h5",
	"a6", "b6", "c6", "d6", "e6", "f6", "g6", "h6",
	"a7", "b7", "c7", "d7", "e7", "f7", "g7", "h7",
	"a8", "b8", "c8", "d8", "e8", "f8", "g8", "h8",
}

// Files
const (
	FileA = iota
	FileB
	FileC
	FileD
	FileE
	FileF
	FileG
	FileH
)

// Ranks
const (
	Rank1 = iota
	Rank2
	Rank3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
)

type Sq int8

// Square returns a square with the given file (0-7) and rank (0-7).
func Square(file, rank int) Sq { return Sq(rank*8 + file) }

// File returns the square's file (0-7).
func (sq Sq) File() int { return int(sq) % 8 }

// Rank returns the square's rank (0-7).
func (sq Sq) Rank() int { return int(sq) / 8 }

// RelativeRank returns the square's rank relative to the given player (0-7).
func (sq Sq) RelativeRank(color int) int {
	if color == White {
		return sq.Rank()
	}
	return 7 - sq.Rank()
}

// String returns the algebraic notation of the square (a1, e5, etc.).
func (sq Sq) String() string {
	if sq == NoSquare {
		return "-"
	}
	return squareNames[sq]
}

func squareFromString(s string) Sq {
	if len(s) != 2 || s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return NoSquare
	}
	return Square(int(s[0])-'a', int(s[1])-'1')
}

// Castling
const (
	queenSide = iota << 1
	kingSide
	WhiteOO  = White | kingSide
	BlackOO  = Black | kingSide
	WhiteOOO = White | queenSide
	BlackOOO = Black | queenSide
)

// Board

// Board represents a regular chess or chess960 position.
type Board struct {
	Piece      [64]Piece // piece placement (NoPiece, WP, BP, WN, BN, ...)
	SideToMove int       // White or Black
	MoveNr     int       // fullmove counter (1-based)
	Rule50     int       // halfmove counter for the 50-move rule (counts from 0-100)
	EpSquare   Sq        // en-passant square (behind capturable pawn)
	CastleSq   [4]Sq     // rooks that can castle; e.g. CastleSq[WhiteOO]
	checkFrom  Sq        // squares the opponent's castling king moved through;
	checkTo    Sq        //      [A1,A1] if opp did not castle last turn.
}

func (b *Board) my(piece int) Piece  { return Piece(b.SideToMove | piece) }
func (b *Board) opp(piece int) Piece { return Piece(b.SideToMove ^ 1 | piece) }

// MustParseFen is like ParseFen, but panics if fen cannot be parsed.
func MustParseFen(fen string) *Board {
	b, err := ParseFen(fen)
	if err != nil {
		panic(err)
	}
	return b
}

// ParseFen initializes a board with the given FEN string. Fields omitted from
// fen will default to the value in the starting position of a regular chess
// game (e.g. 'w' for the side-to-move), so that ParseFen("") returns the
// starting position.
//
// For castling rights both the conventional KkQq can be used as well as file
// letters, for example 'C' for a white rook on the c-file that can castle.
// The latter is sometimes needed for chess960 positions.
func ParseFen(fen string) (b *Board, err error) {
	i, j := 0, 0
	parseError := func(msg interface{}) (*Board, error) {
		return nil, fmt.Errorf("%s·%s: fen error: %s", fen[0:i], fen[i:], msg)
	}
	isSpace := func(c byte) bool {
		return c == ' ' || c == '\t'
	}
	nextField := func(fen string, i, j int, def string) (string, int, int) {
		for i = j; i < len(fen) && isSpace(fen[i]); i++ {
		}
		for j = i; j < len(fen) && !isSpace(fen[j]); j++ {
		}
		if i == j {
			return def, 0, len(def)
		}
		return fen, i, j
	}

	b = new(Board)

	// field 1: pieces
	fen, i, j = nextField(fen, i, j, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR")
	for file, rank := 0, 7; i < j; i++ {
		switch c := rune(fen[i]); c {
		case '/':
			if rank--; rank < 0 {
				return parseError("too many ranks")
			}
			file = 0
		case '1', '2', '3', '4', '5', '6', '7', '8':
			file += int(c - '0')
		default:
			if file > 7 {
				return parseError("too many files")
			}
			piece := pieceFromChar(c)
			if piece == NoPiece {
				return parseError("unexpected character")
			}
			b.Piece[Square(file, rank)] = piece
			file++
		}
	}

	// field 2: side-to-move
	fen, i, j = nextField(fen, i, j, "w")
	switch fen[i:j] {
	case "w":
		b.SideToMove = White
	case "b":
		b.SideToMove = Black
	default:
		return parseError("side-to-move must be 'w' or 'b'")
	}

	// field 3: castling rights
	for i := range b.CastleSq {
		b.CastleSq[i] = NoSquare
	}
	fen, i, j = nextField(fen, i, j, "KQkq")
	if fen[i:j] != "-" {
		for ; i < j; i++ {
			c := int(fen[i])
			b.setCanCastle(c, true)
		}
	}

	// field 4: en-passant square
	fen, i, j = nextField(fen, i, j, "-")
	if fen[i:j] == "-" {
		b.EpSquare = NoSquare
	} else {
		b.EpSquare = squareFromString(fen[i:j])
		if b.EpSquare == NoSquare {
			return parseError("invalid en-passant square")
		}
	}

	// field 5: halfmove counter for the 50-move rule
	fen, i, j = nextField(fen, i, j, "0")
	if b.Rule50, err = strconv.Atoi(fen[i:j]); err != nil {
		return parseError(err)
	}

	// field 6: fullmove counter
	fen, i, j = nextField(fen, i, j, "1")
	if b.MoveNr, err = strconv.Atoi(fen[i:j]); err != nil {
		return parseError(err)
	}

	return b, nil
}

// Fen returns the FEN string (Forsyth-Edwards Notation) of the position.
func (b *Board) Fen() string {
	var fen bytes.Buffer

	// field 1: pieces
	for rank := 7; ; rank-- {
		for file := 0; file <= 8; file++ {
			empty := 0
			for ; file < 8; file++ {
				if b.Piece[Square(file, rank)] != NoPiece {
					break
				}
				empty++
			}
			if empty > 0 {
				fen.WriteRune(rune('0' + empty))
			}
			if file < 8 {
				piece := b.Piece[Square(file, rank)]
				fen.WriteRune(PieceLetters[piece])
			}
		}
		if rank == 0 {
			break
		}
		fen.WriteByte('/')
	}
	fen.WriteByte(' ')

	// field 2: side-to-move
	fen.WriteByte("wb"[b.SideToMove])
	fen.WriteByte(' ')

	// field 3: castling rights - using K/Q for rooks on the h/a files and
	// file letters for other rooks so that for regular chess we always use
	// K/Q. Note that this means that for chess960 K/Q always indicates the
	// rook on the h/a file, not another rook that happens to be on the
	// same side of the king.
	len := fen.Len()
	if sq := b.CastleSq[WhiteOO]; sq != NoSquare {
		if sq == H1 {
			fen.WriteRune('K')
		} else {
			fen.WriteRune(rune('A' + sq.File()))
		}
	}
	if sq := b.CastleSq[WhiteOOO]; sq != NoSquare {
		if sq == A1 {
			fen.WriteRune('Q')
		} else {
			fen.WriteRune(rune('A' + sq.File()))
		}
	}
	if sq := b.CastleSq[BlackOO]; sq != NoSquare {
		if sq == H8 {
			fen.WriteRune('k')
		} else {
			fen.WriteRune(rune('a' + sq.File()))
		}
	}
	if sq := b.CastleSq[BlackOOO]; sq != NoSquare {
		if sq == A8 {
			fen.WriteRune('q')
		} else {
			fen.WriteRune(rune('a' + sq.File()))
		}
	}
	if len == fen.Len() {
		fen.WriteByte('-')
	}
	fen.WriteByte(' ')

	// fields 4-6
	fmt.Fprintf(&fen, "%s %d %d", b.EpSquare, b.Rule50, b.MoveNr)
	return fen.String()
}

// setCanCastle sets or unsets castling rights. c is the file of the rook with
// which to castle ('A'...'H') or 'K'/'Q' for kingside/queenside castling.
// Uppercase for White, lowercase for Black.
func (b *Board) setCanCastle(c int, can bool) {
	var (
		color    int
		sq0, sq1 Sq
	)
	// who's castling?
	switch {
	case c == 'K' || c == 'Q' || (c >= 'A' && c <= 'H'):
		color = White
	case c == 'k' || c == 'q' || (c >= 'a' && c <= 'h'):
		color = Black
	default:
		return
	}
	// find the king
	kingSq := b.find(Piece(color|King), A1, H8)
	if kingSq == NoSquare || kingSq.RelativeRank(color) != Rank1 {
		return
	}
	// determine the range of squares in which to look for the rook
	switch {
	case c == 'Q' || c == 'q':
		sq0 = Square(FileA, kingSq.Rank())
		sq1 = kingSq
	case c == 'K' || c == 'k':
		// sq0 and sq1 in this order! so that if there are two rooks on
		// the kingside, the H-rook is taken.
		sq0 = Square(FileH, kingSq.Rank())
		sq1 = kingSq
	case c >= 'A' && c <= 'H':
		sq0 = Square(c-'A', kingSq.Rank())
		sq1 = sq0
	case c >= 'a' && c <= 'h':
		sq0 = Square(c-'a', kingSq.Rank())
		sq1 = sq0
	}
	// find the rook
	rookSq := b.find(Piece(color|Rook), sq0, sq1)
	if rookSq == NoSquare {
		return
	}
	// set/unset castling right
	wing := kingSide
	if rookSq < kingSq {
		wing = queenSide
	}
	if can {
		b.CastleSq[color|wing] = rookSq
	} else {
		b.CastleSq[color|wing] = NoSquare
	}
}

// MakeMove returns a copy of the Board with move m applied.
func (b Board) MakeMove(m Move) *Board {
	epSquare := b.EpSquare // remember en passant square

	// these are reset by making a move
	b.EpSquare = NoSquare
	b.checkFrom, b.checkTo = A1, A1

	switch {
	case m == NullMove:
		// do nothing
	case b.Piece[m.From] == b.my(King) && b.Piece[m.To] == b.my(Rook): // castling
		wing := kingSide
		if m.To < m.From {
			wing = queenSide
		}
		rf, kf, rt, kt, _, _ := b.castleSquares(wing)
		b.Piece[rf] = NoPiece
		b.Piece[kf] = NoPiece
		b.Piece[rt] = b.my(Rook)
		b.Piece[kt] = b.my(King)
		if kf < kt {
			b.checkFrom, b.checkTo = kf, kt
		} else {
			b.checkFrom, b.checkTo = kt, kf
		}
		b.CastleSq[b.SideToMove|kingSide] = NoSquare
		b.CastleSq[b.SideToMove|queenSide] = NoSquare
		b.Rule50++
	default:
		piece := b.Piece[m.From]
		if piece.Type() == Pawn {
			switch dy := m.To.Rank() - m.From.Rank(); {
			case dy == 2 || dy == -2:
				b.EpSquare = Square(m.From.File(), m.From.Rank()+dy/2)
			case m.To == epSquare:
				// move the captured pawn to the ep-square, so
				// that Rule50 is updated correctly below
				b.Piece[Square(m.To.File(), m.From.Rank())] = NoPiece
				b.Piece[epSquare] = b.opp(Pawn)
			case m.To.RelativeRank(b.SideToMove) == Rank8:
				b.Piece[m.From] = m.Promotion
			}
		}
		// update castling rights
		for i, sq := range b.CastleSq {
			if sq == m.From || sq == m.To {
				b.CastleSq[i] = NoSquare
			}
		}
		if piece.Type() == King {
			b.CastleSq[b.SideToMove|kingSide] = NoSquare
			b.CastleSq[b.SideToMove|queenSide] = NoSquare
		}
		// update the 50-move rule counter
		if piece.Type() == Pawn || b.Piece[m.To] != NoPiece {
			b.Rule50 = 0
		} else {
			b.Rule50++
		}
		// move the piece
		b.Piece[m.To] = b.Piece[m.From]
		b.Piece[m.From] = NoPiece
	}
	// switch side to move
	if b.SideToMove ^= 1; b.SideToMove == White {
		b.MoveNr++
	}
	return &b
}

// find locates a piece in the given range of squares.
func (b *Board) find(piece Piece, sq0, sq1 Sq) Sq {
	dir := Sq(1)
	if sq0 > sq1 {
		dir = -1
	}
	for sq := sq0; sq != sq1+dir; sq += dir {
		if b.Piece[sq] == piece {
			return sq
		}
	}
	return NoSquare
}
