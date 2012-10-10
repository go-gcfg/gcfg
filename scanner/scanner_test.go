// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

import (
	"os"
	"path/filepath"
	"runtime"
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
//FIXME
//	{token.COMMENT, "/* a comment */", special},
//	{token.COMMENT, "// a comment \n", special},

	// Identifiers and basic type literals
//FIXME
//	{token.IDENT, "foobar", literal},
//	{token.IDENT, "a۰۱۸", literal},
//	{token.IDENT, "foo६४", literal},
//	{token.IDENT, "bar９８７６", literal},
//	{token.STRING, "`foobar`", literal},
//	{token.STRING, "`" + `foo
//	                        bar` +
//		"`",
//		literal,
//	},
//	{token.STRING, "`\r`", literal},
//	{token.STRING, "`foo\r\nbar`", literal},

	// Operators and delimiters
//FIXME
//	{token.ASSIGN, "=", operator},
//	{token.LBRACK, "[", operator},
//	{token.RBRACK, "]", operator},
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
	for {
		pos, tok, lit := s.Scan()
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
		_ = pos // checkPos(t, lit, pos, epos) //FIXME
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
		if tok == token.COMMENT && lit[1] == '/' {
			// correct for unaccounted '/n' in //-style comment
			epos.Offset++
			epos.Line++
		}
		index++
		if tok == token.EOF {
			break
		}
	}
	if s.ErrorCount != 0 {
		t.Errorf("found %d errors", s.ErrorCount)
	}
}

var lines = []string{
	// # indicates a semicolon present in the source
	// $ indicates an automatically inserted semicolon
	"",
	"#;",
	"foo$\n",
	"123$\n",
	"1.2$\n",
	"'x'$\n",
	`"x"` + "$\n",
	"`x`$\n",

	"+\n",
	"-\n",
	"*\n",
	"/\n",
	"%\n",

	"&\n",
	"|\n",
	"^\n",
	"<<\n",
	">>\n",
	"&^\n",

	"+=\n",
	"-=\n",
	"*=\n",
	"/=\n",
	"%=\n",

	"&=\n",
	"|=\n",
	"^=\n",
	"<<=\n",
	">>=\n",
	"&^=\n",

	"&&\n",
	"||\n",
	"<-\n",
	"++$\n",
	"--$\n",

	"==\n",
	"<\n",
	">\n",
	"=\n",
	"!\n",

	"!=\n",
	"<=\n",
	">=\n",
	":=\n",
	"...\n",

	"(\n",
	"[\n",
	"{\n",
	",\n",
	".\n",

	")$\n",
	"]$\n",
	"}$\n",
	"#;\n",
	":\n",

	"break$\n",
	"case\n",
	"chan\n",
	"const\n",
	"continue$\n",

	"default\n",
	"defer\n",
	"else\n",
	"fallthrough$\n",
	"for\n",

	"func\n",
	"go\n",
	"goto\n",
	"if\n",
	"import\n",

	"interface\n",
	"map\n",
	"package\n",
	"range\n",
	"return$\n",

	"select\n",
	"struct\n",
	"switch\n",
	"type\n",
	"var\n",

	"foo$//comment\n",
	"foo$//comment",
	"foo$/*comment*/\n",
	"foo$/*\n*/",
	"foo$/*comment*/    \n",
	"foo$/*\n*/    ",

	"foo    $// comment\n",
	"foo    $// comment",
	"foo    $/*comment*/\n",
	"foo    $/*\n*/",
	"foo    $/*  */ /* \n */ bar$/**/\n",
	"foo    $/*0*/ /*1*/ /*2*/\n",

	"foo    $/*comment*/    \n",
	"foo    $/*0*/ /*1*/ /*2*/    \n",
	"foo	$/**/ /*-------------*/       /*----\n*/bar       $/*  \n*/baa$\n",
	"foo    $/* an EOF terminates a line */",
	"foo    $/* an EOF terminates a line */ /*",
	"foo    $/* an EOF terminates a line */ //",

	"package main$\n\nfunc main() {\n\tif {\n\t\treturn /* */ }$\n}$\n",
	"package main$",
}

type segment struct {
	srcline  string // a line of source text
	filename string // filename for current token
	line     int    // line number for current token
}

var segments = []segment{
	// exactly one token per line since the test consumes one token per segment
	{"  line1", filepath.Join("dir", "TestLineComments"), 1},
	{"\nline2", filepath.Join("dir", "TestLineComments"), 2},
	{"\nline3  //line File1.go:100", filepath.Join("dir", "TestLineComments"), 3}, // bad line comment, ignored
	{"\nline4", filepath.Join("dir", "TestLineComments"), 4},
	{"\n//line File1.go:100\n  line100", filepath.Join("dir", "File1.go"), 100},
	{"\n//line File2.go:200\n  line200", filepath.Join("dir", "File2.go"), 200},
	{"\n//line :1\n  line1", "dir", 1},
	{"\n//line foo:42\n  line42", filepath.Join("dir", "foo"), 42},
	{"\n //line foo:42\n  line44", filepath.Join("dir", "foo"), 44},           // bad line comment, ignored
	{"\n//line foo 42\n  line46", filepath.Join("dir", "foo"), 46},            // bad line comment, ignored
	{"\n//line foo:42 extra text\n  line48", filepath.Join("dir", "foo"), 48}, // bad line comment, ignored
	{"\n//line ./foo:42\n  line42", filepath.Join("dir", "foo"), 42},
	{"\n//line a/b/c/File1.go:100\n  line100", filepath.Join("dir", "a", "b", "c", "File1.go"), 100},
}

var unixsegments = []segment{
	{"\n//line /bar:42\n  line42", "/bar", 42},
}

var winsegments = []segment{
	{"\n//line c:\\bar:42\n  line42", "c:\\bar", 42},
	{"\n//line c:\\dir\\File1.go:100\n  line100", "c:\\dir\\File1.go", 100},
}

// Verify that comments of the form "//line filename:line" are interpreted correctly.
func XTestLineComments(t *testing.T) { //FIXME
	segs := segments
	if runtime.GOOS == "windows" {
		segs = append(segs, winsegments...)
	} else {
		segs = append(segs, unixsegments...)
	}

	// make source
	var src string
	for _, e := range segs {
		src += e.srcline
	}

	// verify scan
	var S Scanner
	file := fset.AddFile(filepath.Join("dir", "TestLineComments"), fset.Base(), len(src))
	S.Init(file, []byte(src), nil, 0)
	for _, s := range segs {
		p, _, lit := S.Scan()
		pos := file.Position(p)
		checkPos(t, lit, p, token.Position{
			Filename: s.filename,
			Offset:   pos.Offset,
			Line:     s.line,
			Column:   pos.Column,
		})
	}

	if S.ErrorCount != 0 {
		t.Errorf("found %d errors", S.ErrorCount)
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
//	{"\a", token.ILLEGAL, 0, "illegal character U+0007"},
//	{`#`, token.ILLEGAL, 0, "illegal character U+0023 '#'"},
//	{`…`, token.ILLEGAL, 0, "illegal character U+2026 '…'"},
//	{`""`, token.STRING, 0, ""},
//	{`"`, token.STRING, 0, "string not terminated"},
//	{"``", token.STRING, 0, ""},
//	{"`", token.STRING, 0, "string not terminated"},
//	{"/**/", token.COMMENT, 0, ""},
//	{"/*", token.COMMENT, 0, "comment not terminated"},
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
