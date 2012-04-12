// Package pgn reads chess games from Portable Game Notation (PGN) files.
// (http://www.saremba.de/chessgml/standards/pgn/pgn-complete.htm)
package pgn

import (
	"fmt"
	"github.com/malbrecht/chess"
)

// DB represents a collection of chess games. Its zero value is an empty
// database ready for use.
type DB struct {
	Games []*Game
}

// Game represents a chess game.
type Game struct {
	// Tags holds the PGN tags for the game.
	Tags map[string]string

	// Root is the root node of the main variation of the game. Root.Board
	// is the starting position of the game.
	Root *Node

	// lexer is a lexer set up to parse the movetext section of the game.
	movelex *lexer

	// plies is the number of halfmoves in main variation of the game as
	// counted by the parser while reading it from a PGN file. It is set by
	// the parser upon reading the game, but is not maintained when more
	// nodes are inserted later.
	plies int
}

// Node is an element in the game tree, holding one move. The next move is
// found by following the Next pointer, the previous by following Parent. The
// Variation pointer may point to an alternative list of moves, replacing this
// move. Every variation, including the main line (Game.Root), starts with a
// special root node that repeats the Board of its parent and always has a
// chess.NullMove. It is there to hold any comments preceeding the first move
// of the variation. Use IsRoot to determine whether the node is the root node
// of a variation. Note that following Next never leads to a root node, and
// following Variation always leads to a root node.
type Node struct {
	Parent    *Node        // previous move
	Next      *Node        // next move
	Variation *Node        // an alternative to this move
	Move      chess.Move   // this move
	Board     *chess.Board // position after Move
	Comment   []string     // comment paragraphs on the move
	Nags      []Nag        // annotations
}

// NewGame initializes a new chess game. The starting position of the game, if
// not the default, should be passed as the "FEN" tag in tags. An error is
// returned if the "FEN" tag is specified but cannot be parsed.
func NewGame(tags map[string]string) (*Game, error) {
	board, err := chess.ParseFen(tags["FEN"])
	if err != nil {
		return nil, fmt.Errorf("FEN tag: %s", err)
	}
	g := &Game{
		Tags: tags,
		Root: &Node{Board: board},
	}
	return g, nil
}

// Plies returns the number of halfmoves in the main line. This works even if
// the game was read from a PGN file and ParseMoves has not yet been called.
func (g *Game) Plies() int {
	if g.Root.Next == nil {
		return g.plies
	}
	plies := 0
	for n := g.Root.Next; n != nil; n = n.Next {
		plies++
	}
	return plies
}

// Insert adds a node to the game tree, as a child of n. The new node is
// returned so that consecutive moves can be added like
//     n := game.Root
//     n = n.Insert(m1)
//     n = n.Insert(m2)
//     n = n.Insert(m3)
func (n *Node) Insert(move chess.Move) *Node {
	n.Next = &Node{
		Parent: n,
		Move:   move,
		Board:  n.Board.MakeMove(move),
	}
	return n.Next
}

// NewVariation creates a new variation on n, returning the root node of that
// variation.
func (n *Node) NewVariation() *Node {
	var v *Node
	for v = n; v.Variation != nil; v = v.Variation.Next {
		if v.Variation.Next == nil {
			// Empty variation. Replace with the root of the new
			// variation.
			break
		}
	}
	v.Variation = &Node{
		Parent: n.Parent,
		Board:  n.Parent.Board,
	}
	return v.Variation
}

// Variations returns the list of variations for this node as a slice.
func (n *Node) Variations() []*Node {
	if n.Parent != nil && n.Parent.IsRoot() && n.Parent.Parent != nil {
		// Don't list variations for the first move of a variation, as
		// those variations would have already been listed for a
		// previous node.
		return nil
	}
	var vs []*Node
	for v := n.Variation; v != nil; v = v.Next.Variation {
		if v.Next == nil {
			break // empty variation
		}
		vs = append(vs, v)
	}
	return vs
}

// IsRoot returns whether the node is the root node of a variation.
func (n *Node) IsRoot() bool {
	return n.Parent == nil || n.Parent.Next != n
}

// AddNag adds a NAG to the move.
func (n *Node) AddNag(nag Nag) {
	// don't add duplicates
	for _, x := range n.Nags {
		if nag == x {
			return
		}
	}
	n.Nags = append(n.Nags, nag)
}

// DropNag removes a NAG from the move.
func (n *Node) DropNag(nag Nag) {
	nags := n.Nags
	for i, x := range nags {
		if nag == x {
			nags[i] = nags[len(nags)-1]
			nags = nags[:len(nags)-1]
			return
		}
	}

}

// Parse reads PGN games from a PGN file into the database. Only the tag
// section of each game is loaded, use ParseMoves on each individual game to
// parse the movetext. Parse returns a list of encountered ParseErrors.
func (d *DB) Parse(text string) []error {
	var errs []error
	p := &parser{lex: newLexer(text, 1)}
	for {
		game, err := p.readGame()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if game == nil {
			break
		}
		d.Games = append(d.Games, game)
	}
	return errs
}

// ParseMoves parses the movetext section of the game, generating the game tree
// in game.Root.
func (d *DB) ParseMoves(game *Game) error {
	if game.movelex == nil {
		return nil
	}
	p := &parser{lex: game.movelex}
	oldroot := *game.Root
	if err := p.parseMoves(game.Root); err != nil {
		game.Root = &oldroot
		return err
	}
	game.movelex = nil
	return nil
}
