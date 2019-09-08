package chess

import (
	"bytes"
	"errors"
	"strings"
)

type Move struct {
	From      Sq
	To        Sq
	Promotion Piece
}

var NullMove = Move{}

// isLegal checks the legality of a pseudo-legal move.
func (m Move) isLegal(b *Board) bool {
	b = b.MakeMove(m)
	_, illegal := b.pseudoLegalMoves()
	return !illegal
}

// ParseMove parses a move in algebraic notation. The parser is forgiving and
// will accept varying forms of algebraic notation, including slightly
// incorrect notations (for instance with uncapitalized piece characters).
// Examples: e4, Bb5, cxd3, O-O, 0-0-0, Rae1+, f8=Q, f8/Q, e2-e4, Bf1-b5, e2e4,
// f1b5, e1g1 (castling), f7f8q.
func (b *Board) ParseMove(s string) (Move, error) {
	if s == "--" {
		return NullMove, nil
	}
	var (
		f0, r0    = -1, -1 // from file and rank
		f1, r1    = -1, -1 // to file and rank
		piece     = NoPiece
		promotion = NoPiece
		castle    = -1
		err       = errors.New("invalid move")
	)

	if len(s) < 2 {
		return NullMove, err
	}
	switch {
	case strings.HasPrefix(s, "O-O-O") || strings.HasPrefix(s, "0-0-0"):
		castle = queenSide
	case strings.HasPrefix(s, "O-O") || strings.HasPrefix(s, "0-0"):
		castle = kingSide
	default:
		// The first character may specify the piece type. Lower case
		// piece letters are also accepted. For a 'b' we guess whether
		// it is 'b'ishop or 'b'-file. "bc3" will be interpreted as
		// Bc3, but "b3c4" as b3-c4, not B3c4.
		if p := pieceFromChar(rune(s[0])); p != NoPiece {
			if s[0] != 'b' || (len(s) > 2 && s[1] >= 'a' && s[1] <= 'h') {
				piece = p.Type()
				s = s[1:]
			}
		}
		// Scan for file/rank characters and a promotion piece. A 'b'
		// is always interpreted as a Bishop promotion at first, and
		// reinterpreted as the b-file if more file/rank characters are
		// found.
		for _, c := range s {
			if promotion == Bishop &&
				((c >= 'a' && c <= 'h') || (c >= '1' && c <= '8')) {
				f0, f1 = f1, FileB
				promotion = NoPiece
			}
			switch c {
			case 'b', 'n', 'r', 'q', 'B', 'N', 'R', 'Q':
				promotion = pieceFromChar(c).Type()
			case 'a', 'c', 'd', 'e', 'f', 'g', 'h':
				f0, f1 = f1, int(c-'a')
			case '1', '2', '3', '4', '5', '6', '7', '8':
				r0, r1 = r1, int(c-'1')
			}
		}
		// If the piece type is unknown, because it is not specified
		// and the from-square is unknown, then it must be a pawn (e.g.
		// e4, cxd5).
		if piece == NoPiece && (f0 == -1 || r0 == -1) {
			piece = Pawn
		}
		// Recognize castling as a king either moving two squares, or
		// capturing its own rook.
		if f0 != -1 && f1 != -1 && r0 != -1 && r1 != -1 {
			from, to := Square(f0, r0), Square(f1, r1)
			if b.Piece[from] == b.my(King) && (b.Piece[to] == b.my(Rook) ||
				to == from+2 || to == from-2) {
				if to < from {
					castle = queenSide
				} else {
					castle = kingSide
				}
			}
		}
	}
	// Set f0, r0, f1, r1 for a castling move.
	if castle != -1 {
		rook, king, _, _, _, _ := b.castleSquares(castle)
		if rook == NoSquare || king == NoSquare {
			return NullMove, err
		}
		f0, r0, f1, r1 = king.File(), king.Rank(), rook.File(), rook.Rank()
	}
	// Find the one move matching the parsed files, ranks, piece type and
	// promotion.
	move := NullMove
	moves, _ := b.pseudoLegalMoves()
	for _, m := range moves {
		if (piece == NoPiece || b.Piece[m.From].Type() == piece) &&
			(f0 == -1 || f0 == m.From.File()) &&
			(r0 == -1 || r0 == m.From.Rank()) &&
			(f1 == -1 || f1 == m.To.File()) &&
			(r1 == -1 || r1 == m.To.Rank()) &&
			m.Promotion.Type() == promotion &&
			m.isLegal(b) {
			// the move matches
			if move != NullMove {
				return NullMove, err // ambiguous move
			}
			move = m
		}
	}
	if move == NullMove {
		return NullMove, err
	}
	return move, nil
}

// San returns the move in Universal Chess Interface Notation.
func (m Move) Uci(b *Board) string {
	return m.interfaceNotation(b, PieceLetters)
}

// San returns the move in Standard Algebraic Notation.
func (m Move) San(b *Board) string {
	return m.algebraicNotation(b, PieceLetters)
}

// Fan is like San but uses figurines.
func (m Move) Fan(b *Board) string {
	return m.algebraicNotation(b, Figurines)
}

func (m Move) interfaceNotation(b *Board, pieceLetters []rune) string {
	if m == NullMove {
		return "0000"
	}
	var buf bytes.Buffer
	buf.WriteRune(rune('a' + m.From.File()))
	buf.WriteRune(rune('1' + m.From.Rank()))
	buf.WriteRune(rune('a' + m.To.File()))
	buf.WriteRune(rune('1' + m.To.Rank()))
	if m.Promotion != NoPiece {
		buf.WriteRune(pieceLetters[m.Promotion.Type()])
	}
	return buf.String()
}

func (m Move) algebraicNotation(b *Board, pieceLetters []rune) string {
	if m == NullMove {
		return "--"
	}
	var buf bytes.Buffer
	switch piece := b.Piece[m.From].Type(); {
	case piece == King && b.Piece[m.To] == b.my(Rook):
		if m.From < m.To {
			buf.WriteString("O-O")
		} else {
			buf.WriteString("O-O-O")
		}
	default:
		var disambiguateByFile, disambiguateByRank bool
		isCapture := b.Piece[m.To] != NoPiece
		switch piece {
		case Pawn:
			isCapture = m.From.File() != m.To.File()
			disambiguateByFile = isCapture
		case Knight, Bishop, Rook, Queen:
			moves, _ := b.pseudoLegalMoves()
			for _, n := range moves {
				if n.To == m.To && n.From != m.From &&
					b.Piece[n.From] == b.Piece[m.From] &&
					n.isLegal(b) {
					if n.From.File() != m.From.File() {
						disambiguateByFile = true
					} else {
						disambiguateByRank = true
					}
				}
			}
		}
		if piece != Pawn {
			buf.WriteRune(pieceLetters[piece])
		}
		if disambiguateByFile {
			buf.WriteRune(rune('a' + m.From.File()))
		}
		if disambiguateByRank {
			buf.WriteRune(rune('1' + m.From.Rank()))
		}
		if isCapture {
			buf.WriteRune('x')
		}
		buf.WriteRune(rune('a' + m.To.File()))
		buf.WriteRune(rune('1' + m.To.Rank()))

		if m.Promotion != NoPiece {
			buf.WriteRune('=')
			buf.WriteRune(pieceLetters[m.Promotion.Type()])
		}
	}
	check, mate := b.MakeMove(m).IsCheckOrMate()
	if check {
		if mate {
			buf.WriteRune('#')
		} else {
			buf.WriteRune('+')
		}
	}
	return buf.String()
}
