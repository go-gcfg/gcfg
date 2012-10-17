// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

import (
	"os"
	"strings"
	"testing"
)

import (
	"code.google.com/p/gcfg/token"
)

var fset = token.NewFileSet()

const /* class */ (
	special = iota
	literal
	operator
)

func tokenclass(tok token.Token) int {
	switch {
	case tok.IsLiteral():
		return literal
	case tok.IsOperator():
		return operator
	}
	return special
}

type elt struct {
	tok   token.Token
	lit   string
	class int
}

var tokens = [...]elt{
	// Special tokens
	{token.EOL, "\n", special},

	{token.COMMENT, "; a comment \n", special},
	{token.COMMENT, "# a comment \n", special},

	// Identifiers and basic type literals
//FIXME
	{token.IDENT, "foobar", literal},
	{token.IDENT, "a۰۱۸", literal},
	{token.IDENT, "foo६४", literal},
	{token.IDENT, "bar９８７６", literal},
	{token.IDENT, "foo-bar", literal},
	{token.STRING, `"foobar"`, literal},
	{token.STRING, `"\n"`, literal},
	{token.STRING, `"\""`, literal},
//	{token.STRING, "`" + `foo
//	                        bar` +
//		"`",
//		literal,
//	},
//	{token.STRING, "`\r`", literal},
//	{token.STRING, "`foo\r\nbar`", literal},

	// Operators and delimiters
	{token.ASSIGN, "=", operator},
	{token.LBRACK, "[", operator},
	{token.RBRACK, "]", operator},
}

const whitespace = "  \t  \n\n\n" // to separate tokens

var source = func() []byte {
	var src []byte
	for _, t := range tokens {
		src = append(src, t.lit...)
		src = append(src, whitespace...)
	}
	return src
}()

func newlineCount(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

func checkPos(t *testing.T, lit string, p token.Pos, expected token.Position) {
	pos := fset.Position(p)
	if pos.Filename != expected.Filename {
		t.Errorf("bad filename for %q: got %s, expected %s", lit, pos.Filename, expected.Filename)
	}
	if pos.Offset != expected.Offset {
		t.Errorf("bad position for %q: got %d, expected %d", lit, pos.Offset, expected.Offset)
	}
	if pos.Line != expected.Line {
		t.Errorf("bad line for %q: got %d, expected %d", lit, pos.Line, expected.Line)
	}
	if pos.Column != expected.Column {
		t.Errorf("bad column for %q: got %d, expected %d", lit, pos.Column, expected.Column)
	}
}

// Verify that calling Scan() provides the correct results.
func TestScan(t *testing.T) {
	// make source
	src_linecount := newlineCount(string(source))
	whitespace_linecount := newlineCount(whitespace)

	// error handler
	eh := func(_ token.Position, msg string) {
		t.Errorf("error handler called (msg = %s)", msg)
	}

	// verify scan
	var s Scanner
	s.Init(fset.AddFile("", fset.Base(), len(source)), source, eh, ScanComments)
	index := 0
	// epos is the expected position
	epos := token.Position{
		Filename: "",
		Offset:   0,
		Line:     1,
		Column:   1,
	}
	pos, tok, lit := s.Scan()
outer:
	for {
		if lit == "" {
			// no literal value for non-literal tokens
			lit = tok.String()
		}
		e := elt{token.EOF, "", special}
		if index < len(tokens) {
			e = tokens[index]
		}
		if tok == token.EOF {
			lit = "<EOF>"
			epos.Line = src_linecount
			epos.Column = 2
		}
		checkPos(t, lit, pos, epos)
		if tok != e.tok {
			t.Errorf("bad token for %q: got %s, expected %s", lit, tok, e.tok)
		}
		if e.tok.IsLiteral() {
			// no CRs in raw string literals
			elit := e.lit
			if elit[0] == '`' {
				elit = string(stripCR([]byte(elit)))
				epos.Offset += len(e.lit) - len(lit) // correct position
			}
			if lit != elit {
				t.Errorf("bad literal for %q: got %q, expected %q", lit, lit, elit)
			}
		}
		if tokenclass(tok) != e.class {
			t.Errorf("bad class for %q: got %d, expected %d", lit, tokenclass(tok), e.class)
		}
		epos.Offset += len(lit) + len(whitespace)
		epos.Line += newlineCount(lit) + whitespace_linecount
		if tok == token.COMMENT && strings.HasSuffix(e.lit, "\n") {
			epos.Offset++
			epos.Line++
		}
		index++
		if tok == token.EOF {
			break
		}
		// skip EOLs
		for {
			pos, tok, lit = s.Scan()
			if tok != token.EOL {
				continue outer
			}
		}
	}
	if s.ErrorCount != 0 {
		t.Errorf("found %d errors", s.ErrorCount)
	}
}

// Verify that initializing the same scanner more then once works correctly.
func XTestInit(t *testing.T) { //FIXME
	var s Scanner

	// 1st init
	src1 := "\nname = value"
	f1 := fset.AddFile("src1", fset.Base(), len(src1))
	s.Init(f1, []byte(src1), nil, 0)
	if f1.Size() != len(src1) {
		t.Errorf("bad file size: got %d, expected %d", f1.Size(), len(src1))
	}
	s.Scan()              // \n
	s.Scan()              // name
	_, tok, _ := s.Scan() // =
	if tok != token.ASSIGN {
		t.Errorf("bad token: got %s, expected %s", tok, token.ASSIGN)
	}

	// 2nd init
	src2 := "[section]"
	f2 := fset.AddFile("src2", fset.Base(), len(src2))
	s.Init(f2, []byte(src2), nil, 0)
	if f2.Size() != len(src2) {
		t.Errorf("bad file size: got %d, expected %d", f2.Size(), len(src2))
	}
	_, tok, _ = s.Scan() // [
	if tok != token.LBRACK {
		t.Errorf("bad token: got %s, expected %s", tok, token.LBRACK)
	}

	if s.ErrorCount != 0 {
		t.Errorf("found %d errors", s.ErrorCount)
	}
}

func XTestStdErrorHander(t *testing.T) { //FIXME
	const src = "@\n" + // illegal character, cause an error
		"@ @\n" + // two errors on the same line
		"//line File2:20\n" +
		"@\n" + // different file, but same line
		"//line File2:1\n" +
		"@ @\n" + // same file, decreasing line number
		"//line File1:1\n" +
		"@ @ @" // original file, line 1 again

	var list ErrorList
	eh := func(pos token.Position, msg string) { list.Add(pos, msg) }

	var s Scanner
	s.Init(fset.AddFile("File1", fset.Base(), len(src)), []byte(src), eh, 0)
	for {
		if _, tok, _ := s.Scan(); tok == token.EOF {
			break
		}
	}

	if len(list) != s.ErrorCount {
		t.Errorf("found %d errors, expected %d", len(list), s.ErrorCount)
	}

	if len(list) != 9 {
		t.Errorf("found %d raw errors, expected 9", len(list))
		PrintError(os.Stderr, list)
	}

	list.Sort()
	if len(list) != 9 {
		t.Errorf("found %d sorted errors, expected 9", len(list))
		PrintError(os.Stderr, list)
	}

	list.RemoveMultiples()
	if len(list) != 4 {
		t.Errorf("found %d one-per-line errors, expected 4", len(list))
		PrintError(os.Stderr, list)
	}
}

type errorCollector struct {
	cnt int            // number of errors encountered
	msg string         // last error message encountered
	pos token.Position // last error position encountered
}

func checkError(t *testing.T, src string, tok token.Token, pos int, err string) {
	var s Scanner
	var h errorCollector
	eh := func(pos token.Position, msg string) {
		h.cnt++
		h.msg = msg
		h.pos = pos
	}
	s.Init(fset.AddFile("", fset.Base(), len(src)), []byte(src), eh, ScanComments)
	_, tok0, _ := s.Scan()
	_, tok1, _ := s.Scan()
	if tok0 != tok {
		t.Errorf("%q: got %s, expected %s", src, tok0, tok)
	}
	if tok1 != token.EOF {
		t.Errorf("%q: got %s, expected EOF", src, tok1)
	}
	cnt := 0
	if err != "" {
		cnt = 1
	}
	if h.cnt != cnt {
		t.Errorf("%q: got cnt %d, expected %d", src, h.cnt, cnt)
	}
	if h.msg != err {
		t.Errorf("%q: got msg %q, expected %q", src, h.msg, err)
	}
	if h.pos.Offset != pos {
		t.Errorf("%q: got offset %d, expected %d", src, h.pos.Offset, pos)
	}
}

var errors = []struct {
	src string
	tok token.Token
	pos int
	err string
}{
//FIXME
	{"\a", token.ILLEGAL, 0, "illegal character U+0007"},
	{"/", token.ILLEGAL, 0, "illegal character U+002F '/'"},
	{"_", token.ILLEGAL, 0, "illegal character U+005F '_'"},
	{`…`, token.ILLEGAL, 0, "illegal character U+2026 '…'"},
	{`""`, token.STRING, 0, ""},
	{`"`, token.STRING, 0, "string not terminated"},
	{`"\z"`, token.STRING, 2, "unknown escape sequence"},
	{`"\a"`, token.STRING, 2, "unknown escape sequence"},
	{`"\b"`, token.STRING, 2, "unknown escape sequence"},
	{`"\f"`, token.STRING, 2, "unknown escape sequence"},
	{`"\r"`, token.STRING, 2, "unknown escape sequence"},
	{`"\t"`, token.STRING, 2, "unknown escape sequence"},
	{`"\v"`, token.STRING, 2, "unknown escape sequence"},
	{`"\0"`, token.STRING, 2, "unknown escape sequence"},
//	{"`", token.STRING, 0, "string not terminated"},
//	{"\"abc\x00def\"", token.STRING, 4, "illegal character NUL"},
//	{"\"abc\x80def\"", token.STRING, 4, "illegal UTF-8 encoding"},
}

func TestScanErrors(t *testing.T) {
	for _, e := range errors {
		checkError(t, e.src, e.tok, e.pos, e.err)
	}
}

func BenchmarkScan(b *testing.B) {
	b.StopTimer()
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(source))
	var s Scanner
	b.StartTimer()
	for i := b.N - 1; i >= 0; i-- {
		s.Init(file, source, nil, ScanComments)
		for {
			_, tok, _ := s.Scan()
			if tok == token.EOF {
				break
			}
		}
	}
}
