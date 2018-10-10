package pgn

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// item represents a token or text string returned from the scanner.
type item struct {
	typ itemType
	val string
}

// itemType identifies the type of lex items.
type itemType int

const (
	itemNone itemType = iota
	itemEOF
	itemLBracket   // '['
	itemRBracket   // ']'
	itemLParen     // '('
	itemRParen     // ')'
	itemSymbol     // a tag name ('Event') or a move ('Bxe5+')
	itemString     // quoted string (includes quotes)
	itemComment    // block comment (includes braces); line comments are ignored
	itemAnnotation // annotation: '!' '?!' '$1' '$2' etc
	itemResult     // '1-0' '0-1' '1/2-1/2' '*'
	itemMoveNumber // move number
	itemDots       // dots following a move number
)

var itemNames = map[itemType]string{
	itemNone:       "<none>",
	itemEOF:        "<EOF>",
	itemLBracket:   "'['",
	itemRBracket:   "']'",
	itemLParen:     "'('",
	itemRParen:     "')'",
	itemSymbol:     "<symbol>",
	itemString:     "<string>",
	itemComment:    "<comment>",
	itemAnnotation: "<annotation>",
	itemResult:     "<result>",
	itemMoveNumber: "<movenr>",
	itemDots:       "<dots>",
}

func (i itemType) String() string {
	return itemNames[i]
}

const eof = -1

// lexer holds the state of the scanner.
type lexer struct {
	input   string // the input being scanned
	pos     int    // current position in the input
	line    int    // current line in the input
	start   int    // start position of the next item
	emitted item   // the item being emitted
}

func newLexer(input string, lineoff int) *lexer {
	l := &lexer{
		input: input,
		line:  lineoff,
	}
	return l
}

// peek returns the next rune in the input.
func (l *lexer) peek() rune {
	r, _ := l.nextRune()
	return r
}

// next consumes the next rune in the input.
func (l *lexer) next() rune {
	r, size := l.nextRune()
	l.pos += size
	if r == '\n' {
		l.line++
	}
	return r
}

func (l *lexer) nextRune() (r rune, size int) {
	if l.pos >= len(l.input) {
		return eof, 0
	}
	return utf8.DecodeRuneInString(l.input[l.pos:])
}

// coords returns the line and column number of the current position+offset.
func (l *lexer) coords(offset int) (line, col int) {
	pos := l.pos + offset
	line = l.line - strings.Count(l.input[pos:l.pos], "\n")
	for col = 1; col <= pos; col++ {
		if l.input[pos-col] == '\n' {
			break
		}
	}
	return line, col
}

// panicf panics with a lexPanic to be caught by the parser.
func (l *lexer) panicf(format string, args ...interface{}) {
	err := lexPanic(fmt.Sprintf(format, args...))
	panic(err)
}

// emit turns the pending input into an item.
func (l *lexer) emit(t itemType) {
	l.emitted = item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore discards the pending input.
func (l *lexer) ignore() {
	l.start = l.pos
}

// acceptRun consumes a run of runes from the runes set.
func (l *lexer) acceptRun(runes string) {
	for strings.IndexRune(runes, l.peek()) >= 0 {
		l.next()
	}
}

// find consumes a run of runes not in the runes set.
func (l *lexer) find(runes string) bool {
	for {
		if r := l.next(); r == eof {
			return false
		} else if strings.IndexRune(runes, r) >= 0 {
			return true
		}
	}
	panic("unreachable")
}

// recover tries to find the next game by scanning until an empty line.
func (l *lexer) recover() {
loop:
	for {
		switch l.next() {
		case eof:
			break loop
		case '\n':
			l.acceptRun(" \t\r")
			if l.next() == '\n' {
				break loop
			}
		}
	}
	l.ignore()
}

// item returns the next item from the input.
func (l *lexer) item() item {
	l.emitted = item{}
	for l.emitted == (item{}) {
		switch r := l.next(); r {
		case eof:
			l.emit(itemEOF)
		case ' ', '\t', '\v', '\r', '\n':
			l.acceptRun(" \t\v\r\n")
			l.ignore()
		case ';', '%':
			l.find("\n")
			l.ignore()
		case '[':
			l.emit(itemLBracket)
		case ']':
			l.emit(itemRBracket)
		case '(':
			l.emit(itemLParen)
		case ')':
			l.emit(itemRParen)
		case '*':
			l.emit(itemResult)
		case '{':
			if !l.find("}") {
				l.panicf("unclosed block comment")
			}
			l.emit(itemComment)
		case '"':
			l.string()
		case '$':
			l.acceptRun("0123456789")
			if l.pos-l.start < 2 {
				l.panicf("expected digit")
			}
			l.emit(itemAnnotation)
		case '!', '?':
			l.acceptRun("!?")
			l.emit(itemAnnotation)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			l.number()
		case '.':
			l.acceptRun(".")
			l.emit(itemDots)
		default:
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				l.panicf("unexpected character: %#U", r)
			}
			l.acceptRun("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+#=:-")
			l.emit(itemSymbol)
		}
	}
	return l.emitted
}

func (l *lexer) number() {
	// Check if the number is not, in fact, a game result.
	results := [...]string{"1-0", "0-1", "1/2-1/2"}
	for _, result := range results {
		if strings.HasPrefix(l.input[l.start:], result) {
			l.pos = l.start + len(result)
			l.emit(itemResult)
			return
		}
	}
	l.acceptRun("0123456789")
	l.emit(itemMoveNumber)
}

func (l *lexer) string() {
	for {
		switch l.next() {
		case '\\':
			l.next()
		case eof, '\n':
			l.panicf("unclosed quoted string")
		case '"':
			l.emit(itemString)
			return
		}
	}
}
