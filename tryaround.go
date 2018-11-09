package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

var replacementvars = map[string]string{
	"LCG_releases_base": "/cvmfs/lhcb.cern.ch/lib/lcg/releases",
}

var all_libs = map[string]bool{}

type ListsFile struct {
	Functions []*Function `{ @@ }`
}

type Function struct {
	Pos          lexer.Position
	FunctionName string   `@Ident`
	Fargs        []string `"(" { @( Arg | String | Ident ) } ")"`
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		log.Print("No arguments provided")
		os.Exit(4)
	}

	mylexer := lexer.Must(ebnf.New(`
	Comment = "#" { "\u0000"…"\uffff"-"\n"-"\r" } .
	Ident = identchar { identchar } .
	Arg = argchar { argchar } .
	OParenthesis = "(" .
	CParenthesis = ")" .
	String = "\"" { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } "\"" .
	Whitespace = " " | "\t" | "\n" | "\r" .
	EOL = ( "\n" | "\r" ) { "\n" | "\r" } .

	argchar = "_" | "$" | "{" | "}" | "a"…"z" | "0"…"9" | "." | ";" | "-" | "A"…"Z" | "/" .
	identchar = "_" | "a"…"z" | "0"…"9" | "A"…"Z" .
	any = "\u0000"…"\uffff" .
	`))

	parser := participle.MustBuild(&ListsFile{},
		participle.Lexer(mylexer),
		participle.Elide("Comment", "Whitespace"),
	)
	cmakelists := &ListsFile{}
	for _, fname := range flag.Args() {
		filecontent, err := os.Open(fname)
		if err != nil {
			log.Printf("Couldn't open file: %v", err)
			os.Exit(3)
		}
		defer filecontent.Close()
		err = parser.Parse(filecontent, cmakelists)
		if err != nil {
			log.Printf("Parsing failed: %v", err)
			os.Exit(2)
		}
		log.Printf("result is:")
		for _, f := range cmakelists.Functions {
			if f.FunctionName == "set_target_properties" {
				i := 0
				for ; i < len(f.Fargs); i++ {
					if f.Fargs[i] == "PROPERTIES" {
						break
					}
				}
				for i++; i < len(f.Fargs); i += 2 {
					property := f.Fargs[i]
					value := f.Fargs[i+1]
					if property == "REQUIRED_LIBRARIES" {
						for k, v := range replacementvars {
							log.Printf("replacing %s by %s", k, v)
							value = strings.Replace(value, "${"+k+"}", v, -1)
						}
						for _, l := range strings.Split(value, ";") {
							all_libs[l] = true
						}
						log.Printf("%s has value %s", property, value)
					}
				}
			}
		}
	}
	log.Print("found: ")
	for l, _ := range all_libs {
		log.Printf("     - %s", l)
	}

}
