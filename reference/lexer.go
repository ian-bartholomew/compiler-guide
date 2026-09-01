package main

// TokenKind is the category of a lexical token.
type TokenKind int

const (
	TEOF TokenKind = iota
	TInt
	TIdent
	// keywords
	TFunc
	TLet
	TIf
	TElse
	TWhile
	TReturn
	TPrint
	// punctuation
	TLParen
	TRParen
	TLBrace
	TRBrace
	TComma
	TSemicolon
	TAssign
	// operators
	TPlus
	TMinus
	TStar
	TSlash
	TEq
	TNe
	TLt
	TGt
	TLe
	TGe
)

// Token is one lexical unit, with its raw text and source line.
type Token struct {
	Kind TokenKind
	Text string
	Line int
}

var keywords = map[string]TokenKind{
	"func":   TFunc,
	"let":    TLet,
	"if":     TIf,
	"else":   TElse,
	"while":  TWhile,
	"return": TReturn,
	"print":  TPrint,
}

var singles = map[byte]TokenKind{
	'(': TLParen,
	')': TRParen,
	'{': TLBrace,
	'}': TRBrace,
	',': TComma,
	';': TSemicolon,
	'=': TAssign,
	'+': TPlus,
	'-': TMinus,
	'*': TStar,
	'/': TSlash,
	'<': TLt,
	'>': TGt,
}

// lex turns source text into a slice of tokens ending in TEOF.
func lex(src string) []Token {
	var toks []Token
	line := 1
	i, n := 0, len(src)
	for i < n {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '#': // comment to end of line
			for i < n && src[i] != '\n' {
				i++
			}
		case isDigit(c):
			j := i
			for j < n && isDigit(src[j]) {
				j++
			}
			toks = append(toks, Token{TInt, src[i:j], line})
			i = j
		case isAlpha(c):
			j := i
			for j < n && isAlphaNum(src[j]) {
				j++
			}
			text := src[i:j]
			kind := TIdent
			if k, ok := keywords[text]; ok {
				kind = k
			}
			toks = append(toks, Token{kind, text, line})
			i = j
		default:
			// two-character operators first
			if i+1 < n {
				if k, ok := twoChar(src[i], src[i+1]); ok {
					toks = append(toks, Token{k, src[i : i+2], line})
					i += 2
					continue
				}
			}
			k, ok := singles[c]
			if !ok {
				fatalf(line, "unexpected character %q", string(c))
			}
			toks = append(toks, Token{k, string(c), line})
			i++
		}
	}
	toks = append(toks, Token{TEOF, "", line})
	return toks
}

func twoChar(a, b byte) (TokenKind, bool) {
	switch {
	case a == '=' && b == '=':
		return TEq, true
	case a == '!' && b == '=':
		return TNe, true
	case a == '<' && b == '=':
		return TLe, true
	case a == '>' && b == '=':
		return TGe, true
	}
	return TEOF, false
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isAlphaNum(c byte) bool { return isAlpha(c) || isDigit(c) }
