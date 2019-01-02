package main

import (
	"log"
	"os"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

type ListsFile struct {
	Functions []*Function `{ @@ }`
}

type Function struct {
	Pos          lexer.Position
	FunctionName string   `@Ident`
	Fargs        []string `{ @( Arg | String ) } ")"`
}

func parse(fname string) (*ListsFile, error) {
	mylexer := lexer.Must(ebnf.New(`
	Comment = "#" { "\u0000"…"\uffff"-"\n"-"\r" } .
	Ident = identchar { identchar } "(" .
	Arg = argchar { argchar } .
	CParenthesis = ")" .
	String = "\"" { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } "\"" .
	Whitespace = " " | "\t" | "\n" | "\r" .
	EOL = ( "\n" | "\r" ) { "\n" | "\r" } .

	argchar = "_" | "$" | "{" | "}" | "a"…"z" | "0"…"9" | "." | ";" | "-" | "A"…"Z" | "/" | ( "\\" any ) | "+" | ":" | "*" .
	identchar = "_" | "a"…"z" | "0"…"9" | "A"…"Z" .
	any = "\u0000"…"\uffff" .
	`))

	parser := participle.MustBuild(&ListsFile{},
		participle.Lexer(mylexer),
		participle.Elide("Comment", "Whitespace"),
	)
	cmakelists := &ListsFile{}
	filecontent, err := os.Open(fname)
	if err != nil {
		log.Printf("Couldn't open file: %v", err)
		return nil, err
	}
	defer filecontent.Close()
	err = parser.Parse(filecontent, cmakelists)
	if err != nil {
		log.Printf("Parsing failed: %v", err)
		return nil, err
	}
	return cmakelists, nil
}
