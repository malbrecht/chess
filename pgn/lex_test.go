package pgn

import (
	"reflect"
	"testing"
)

type lexTest struct {
	name  string
	input string
	items []item
}

var (
	tEOF = item{itemEOF, ""}
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"spaces", " \t\r", []item{tEOF}},
	{"pragma", "% ignore this line", []item{tEOF}},
	{"line comment", "; line comment", []item{tEOF}},
	{"block comment", "{ block\ncomment }", []item{
		{itemComment, "{ block\ncomment }"},
		tEOF,
	}},
	{"tag", `[Event "casual game"]`, []item{
		{itemLBracket, "["},
		{itemSymbol, "Event"},
		{itemString, `"casual game"`},
		{itemRBracket, "]"},
		tEOF,
	}},
	{"moves", "12. O-O-O Bxe5+ (12... e8=Q)", []item{
		{itemMoveNumber, "12"},
		{itemDots, "."},
		{itemSymbol, "O-O-O"},
		{itemSymbol, "Bxe5+"},
		{itemLParen, "("},
		{itemMoveNumber, "12"},
		{itemDots, "..."},
		{itemSymbol, "e8=Q"},
		{itemRParen, ")"},
		tEOF,
	}},
	{"results", `1-0 0-1 1/2-1/2 *`, []item{
		{itemResult, "1-0"},
		{itemResult, "0-1"},
		{itemResult, "1/2-1/2"},
		{itemResult, "*"},
		tEOF,
	}},
	{"annotations", `$4 $12 Bxe5+? Bxe5+?!`, []item{
		{itemAnnotation, "$4"},
		{itemAnnotation, "$12"},
		{itemSymbol, "Bxe5+"},
		{itemAnnotation, "?"},
		{itemSymbol, "Bxe5+"},
		{itemAnnotation, "?!"},
		tEOF,
	}},
	{"escaped string", `[Event "a\"b"]`, []item{
		{itemLBracket, "["},
		{itemSymbol, "Event"},
		{itemString, `"a\"b"`},
		{itemRBracket, "]"},
		tEOF,
	}},
	// errors
	{"badchar", "[Event \x01]", []item{
		{itemLBracket, "["},
		{itemSymbol, "Event"},
		{itemNone, "unexpected character: U+0001"},
	}},
	{"unclosed string", `"casual game`, []item{
		{itemNone, "unclosed quoted string"},
	}},
	{"unclosed comment", `{ block\ncomment`, []item{
		{itemNone, "unclosed block comment"},
	}},
	{"bad nag", `$a`, []item{
		{itemNone, "expected digit"},
	}},
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest) (items []item) {
	defer func() {
		if e := recover(); e != nil {
			err, ok := e.(lexPanic)
			if !ok {
				panic(e)
			}
			items = append(items, item{itemNone, string(err)})
		}
	}()
	l := newLexer(t.input, 1)
	for {
		item := l.item()
		items = append(items, item)
		if item.typ == itemEOF {
			break
		}
	}
	return
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test)
		if !reflect.DeepEqual(items, test.items) {
			t.Errorf("%s: got\n\t%v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}
