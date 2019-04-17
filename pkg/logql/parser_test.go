package logql

import (
	"strings"
	"testing"
	"text/scanner"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestLex(t *testing.T) {
	for _, tc := range []struct {
		input    string
		expected []int
	}{
		{`{foo="bar"}`, []int{OPEN_BRACE, IDENTIFIER, EQ, STRING, CLOSE_BRACE}},
		{`{ foo = "bar" }`, []int{OPEN_BRACE, IDENTIFIER, EQ, STRING, CLOSE_BRACE}},
		{`{ foo != "bar" }`, []int{OPEN_BRACE, IDENTIFIER, NEQ, STRING, CLOSE_BRACE}},
		{`{ foo =~ "bar" }`, []int{OPEN_BRACE, IDENTIFIER, RE, STRING, CLOSE_BRACE}},
		{`{ foo !~ "bar" }`, []int{OPEN_BRACE, IDENTIFIER, NRE, STRING, CLOSE_BRACE}},
		{`{ foo = "bar", bar != "baz" }`, []int{OPEN_BRACE, IDENTIFIER, EQ, STRING,
			COMMA, IDENTIFIER, NEQ, STRING, CLOSE_BRACE}},
		{`{ foo = "ba\"r" }`, []int{OPEN_BRACE, IDENTIFIER, EQ, STRING, CLOSE_BRACE}},
	} {
		t.Run(tc.input, func(t *testing.T) {
			actual := []int{}
			l := lexer{
				Scanner: scanner.Scanner{
					Mode: scanner.SkipComments | scanner.ScanStrings,
				},
			}
			l.Init(strings.NewReader(tc.input))
			var lval exprSymType
			for {
				tok := l.Lex(&lval)
				if tok == 0 {
					break
				}
				actual = append(actual, tok)
			}
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		in  string
		exp Expr
		err error
	}{
		{
			in:  `{foo="bar"}`,
			exp: &matchersExpr{matchers: []*labels.Matcher{mustNewMatcher(labels.MatchEqual, "foo", "bar")}},
		},
		{
			in:  `{http.url=~"^/admin"}`,
			exp: &matchersExpr{matchers: []*labels.Matcher{mustNewMatcher(labels.MatchRegexp, "http.url", "^/admin")}},
		},
		{
			in:  `{ foo = "bar" }`,
			exp: &matchersExpr{matchers: []*labels.Matcher{mustNewMatcher(labels.MatchEqual, "foo", "bar")}},
		},
		{
			in:  `{ foo != "bar" }`,
			exp: &matchersExpr{matchers: []*labels.Matcher{mustNewMatcher(labels.MatchNotEqual, "foo", "bar")}},
		},
		{
			in:  `{ foo =~ "bar" }`,
			exp: &matchersExpr{matchers: []*labels.Matcher{mustNewMatcher(labels.MatchRegexp, "foo", "bar")}},
		},
		{
			in:  `{ foo !~ "bar" }`,
			exp: &matchersExpr{matchers: []*labels.Matcher{mustNewMatcher(labels.MatchNotRegexp, "foo", "bar")}},
		},
		{
			in: `{ foo = "bar", bar != "baz" }`,
			exp: &matchersExpr{matchers: []*labels.Matcher{
				mustNewMatcher(labels.MatchEqual, "foo", "bar"),
				mustNewMatcher(labels.MatchNotEqual, "bar", "baz"),
			}},
		},
		{
			in: `{foo="bar"} |= "baz"`,
			exp: &matchExpr{
				left:  &matchersExpr{matchers: []*labels.Matcher{mustNewMatcher(labels.MatchEqual, "foo", "bar")}},
				ty:    labels.MatchEqual,
				match: "baz",
			},
		},
		{
			in: `{foo="bar"} |= "baz" |~ "blip" != "flip" !~ "flap"`,
			exp: &matchExpr{
				left: &matchExpr{
					left: &matchExpr{
						left: &matchExpr{
							left:  &matchersExpr{matchers: []*labels.Matcher{mustNewMatcher(labels.MatchEqual, "foo", "bar")}},
							ty:    labels.MatchEqual,
							match: "baz",
						},
						ty:    labels.MatchRegexp,
						match: "blip",
					},
					ty:    labels.MatchNotEqual,
					match: "flip",
				},
				ty:    labels.MatchNotRegexp,
				match: "flap",
			},
		},
		{
			in: `{foo="bar}`,
			err: ParseError{
				msg:  "literal not terminated",
				line: 1,
				col:  6,
			},
		},
		{
			in: `{foo="bar"} |~`,
			err: ParseError{
				msg:  "unexpected end of query, expected string",
				line: 1,
				col:  15,
			},
		},
		{
			in: `{foo="bar"} "foo"`,
			err: ParseError{
				msg:  "unexpected string, expected pipe",
				line: 1,
				col:  13,
			},
		},
	} {
		t.Run(tc.in, func(t *testing.T) {
			ast, err := ParseExpr(tc.in)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.exp, ast)
		})
	}
}