package pgn

import (
	"fmt"
	"strconv"
	"strings"
)

// parser holds the state of the parser.
type parser struct {
	lex      *lexer
	pos      int  // position of current item in input
	item     item // current item
	lastitem item // previous item
}

// ParseError describes a problem parsing a pgn file.
type ParseError struct {
	Line    int
	Col     int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Message)
}

// lexPanic and parsePanic are used to discern panics from the lexer and parser
// from runtime panics.
type (
	lexPanic   string
	parsePanic string
)

func (p *parser) panicf(format string, args ...interface{}) {
	err := parsePanic(fmt.Sprintf(format, args...))
	panic(err)
}

// recover recovers from a lexPanic or parsePanic and produces a ParseError in
// *errp.
func (p *parser) recover(errp *error) {
	err := recover()
	if err == nil {
		return
	}
	var (
		line int
		col  int
		msg  string
	)
	switch v := err.(type) {
	case lexPanic:
		line, col = p.lex.coords(-1)
		msg = string(v)
	case parsePanic:
		line, col = p.lex.coords(p.pos - p.lex.pos)
		msg = string(v)
	default:
		panic(err)
	}
	*errp = &ParseError{
		Line:    line,
		Col:     col,
		Message: msg,
	}
	// Try to continue parsing at the next game in the input.
	p.lex.recover()
	p.item = item{}
}

// next gets the next item from the lexer.
func (p *parser) next() {
	p.lastitem = p.item
	p.pos = p.lex.pos
	p.item = p.lex.item()
}

// accept consumes an item (skipping comments) if it has the requested type.
func (p *parser) accept(typ itemType) bool {
	for p.item.typ == itemComment {
		p.next()
	}
	if p.item.typ != typ {
		return false
	}
	p.next()
	return true
}

// expect is like accept, but panics if the item type does not match.
func (p *parser) expect(typ itemType) item {
	if !p.accept(typ) {
		p.panicf("expected %s, got %s", typ, p.item.typ)
	}
	return p.lastitem
}

// unescape unquotes and unescapes a backslash-escaped PGN string.
func unescape(s string) string {
	return strings.Replace(unquote(s), "\\", "", -1)
}

// unquote removes the first and last character from s, trimming the result.
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	return strings.TrimSpace(s[1 : len(s)-1])
}

// readGame reads the game information of the next game in the input file. It
// returns nil,nil if no more games are available.
func (p *parser) readGame() (game *Game, err error) {
	defer p.recover(&err)
	if p.item == (item{}) {
		p.next()
	}
	if p.accept(itemEOF) {
		return nil, nil
	}
	var (
		mtext0    = p.pos
		mtextline = p.lex.line
		tags      = make(map[string]string)
	)
	for p.accept(itemLBracket) {
		tag := p.expect(itemSymbol).val
		val := p.expect(itemString).val
		tags[tag] = unescape(val)
		p.expect(itemRBracket)
		// Remember where the movetext starts. Maintaining this inside
		// the loop ensures that initial comments, which will be
		// skipped by the next accept() call, are included in the
		// movetext.
		mtext0 = p.pos
		mtextline = p.lex.line
	}
	if len(tags) == 0 {
		p.panicf("no game tags found")
	}
	// Parsing and validating the moves in the movetext section is
	// postponed until parseMoves is called. Here we just quickly scan the
	// movetext to get some additional game information: the number of
	// moves in the main line and the game result in case it was not
	// already present in the tags section.
	plies := 0
	variant := 0
loop:
	for {
		switch p.item.typ {
		case itemLParen:
			variant++
		case itemRParen:
			variant--
		case itemSymbol:
			if variant == 0 {
				plies++
			}
		case itemResult:
			if result, ok := tags["Result"]; !ok {
				tags["Result"] = p.item.val
			} else if result != p.item.val {
				p.panicf("game result %q differs from Result tag %q", p.item.val, result)
			}
		case itemLBracket, itemEOF:
			break loop
		}
		p.next()
	}
	mtext1 := p.pos
	if tags["Result"] == "" {
		tags["Result"] = "*"
	}
	g, err := NewGame(tags)
	if err != nil {
		p.panicf("%s", err)
	}
	g.plies = plies
	g.movelex = newLexer(p.lex.input[mtext0:mtext1], mtextline)
	return g, nil
}

// parseMoves parses a movetext section, knowing that p.lex has been set up to
// lex a single such section.
func (p *parser) parseMoves(root *Node) (err error) {
	defer p.recover(&err)
	if p.item == (item{}) {
		p.next()
	}
	p.variation(root, 0)
	return nil
}

// variation parses recursive variations (lists of moves).
func (p *parser) variation(node *Node, level int) {
	for {
		switch p.item.typ {
		case itemSymbol: // a move
			move, err := node.Board.ParseMove(p.item.val)
			if err != nil {
				p.panicf("%q: %s", p.item.val, err)
			}
			node = node.Insert(move)
		case itemComment:
			node.Comment = append(node.Comment, unquote(p.item.val))
		case itemAnnotation:
			node.AddNag(p.nag(p.item.val))
		case itemLParen:
			if node.IsRoot() {
				p.panicf("variation without a preceeding move")
			}
			p.next()
			p.variation(node.NewVariation(), level+1)
		case itemRParen:
			if level == 0 {
				p.panicf("unexpected right parenthesis")
			}
			return
		case itemEOF, itemLBracket:
			if level != 0 {
				p.panicf("%d unclosed variations", level)
			}
			return
		case itemMoveNumber, itemDots, itemResult:
			// ignore
		default:
			p.panicf("unexpected token: %s", p.item.typ)
		}
		p.next()
	}
}

// nag extracts a Nag from s.
func (p *parser) nag(s string) Nag {
	if len(s) >= 2 && s[0] == '$' {
		if n, err := strconv.Atoi(s[1:]); err == nil {
			return Nag(n)
		}
	} else {
		switch s {
		case "!":
			return 1
		case "?":
			return 2
		case "!!":
			return 3
		case "??":
			return 4
		case "!?":
			return 5
		case "?!":
			return 6
		}
	}
	p.panicf("%q: invalid annotation", s)
	panic("unreachable")
}
