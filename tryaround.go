package main

import (
	_ "flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

var replacementvars = map[string]string{
	"LCG_releases_base": "/cvmfs/lhcb.cern.ch/lib/lcg/releases",
}

var all_libs = map[string]bool{}
var upperCaseL = map[string]bool{}
var lowerCaseL = map[string]bool{}
var targetLs = map[string]bool{}
var linkerlibs = map[string]bool{}

// project's linker lib -> dependencies got resolved

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

	argchar = "_" | "$" | "{" | "}" | "a"…"z" | "0"…"9" | "." | ";" | "-" | "A"…"Z" | "/" | ( "\\" any ) | "+" .
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

func main() {
	platformconfigpath := "/cvmfs/lhcbdev.cern.ch/nightlies/lhcb-head/latest/REC/REC_HEAD/InstallArea/x86_64+avx2+fma-centos7-gcc7-opt/cmake/RecPlatformConfig.cmake"
	platformconfig, err := parse(platformconfigpath)
	if err != nil {
		log.Printf("Couldn't open project config: %v", err)
		os.Exit(7)
	}
	for _, funccall := range platformconfig.Functions {
		if funccall.FunctionName == "set(" && funccall.Fargs[0] == "Rec_LINKER_LIBRARIES" {
			for _, linklib := range strings.Split(funccall.Fargs[1], ";") {
				linkerlibs[linklib] = false
			}
		}
	}

	exportfiles, err := filepath.Glob("/cvmfs/lhcbdev.cern.ch/nightlies/lhcb-head/latest/REC/REC_HEAD/InstallArea/x86_64+avx2+fma-centos7-gcc7-opt/cmake/*Export.cmake")
	if err != nil {
		log.Printf("couldn't glob export files: %v", err)
		os.Exit(9)
	}
	for _, exportfile := range exportfiles {
		cmakelists, err := parse(exportfile)
		if err != nil {
			log.Printf("%v", err)
			os.Exit(3)
		}

		for _, f := range cmakelists.Functions {
			if f.FunctionName == "set_target_properties(" {
				i := 0
				var loclibs []string
				for ; i < len(f.Fargs); i++ {
					if f.Fargs[i] == "PROPERTIES" {
						break
					} else {
						loclibs = append(loclibs, f.Fargs[i])
					}
				}
				need_this := false
				for _, loclib := range loclibs {
					if _, found := linkerlibs[loclib]; found {
						linkerlibs[loclib] = true
						need_this = true
					}
				}
				if !need_this {
					break
				}
				for i++; i < len(f.Fargs); i += 2 {
					property := f.Fargs[i]
					value := f.Fargs[i+1]
					if value[0] == '"' && value[len(value)-1] == '"' {
						value = value[1 : len(value)-1]
					}
					if property == "REQUIRED_LIBRARIES" {
						for k, v := range replacementvars {
							log.Printf("replacing %s by %s", k, v)
							value = strings.Replace(value, "${"+k+"}", v, -1)
						}
						for _, l := range strings.Split(value, ";") {
							all_libs[l] = true
						}
					}
				}
			}
		}
	}
	for l, r := range linkerlibs {
		if !r {
			log.Printf("WARNING: couldn't resolve link deps for %s", l)
		}
	}
	for l, _ := range all_libs {
		if _, err := os.Stat(l); !os.IsNotExist(err) {
			dir, file := filepath.Split(l)
			validateme := file[3 : len(file)-3]
			if "lib"+validateme+".so" != file {
				log.Print("PANIC")
				os.Exit(777)
			}
			upperCaseL[dir] = true
			lowerCaseL[validateme] = true
		} else {
			targetLs[l] = true
		}
	}
	for l, _ := range upperCaseL {
		log.Printf("dir: %s", l)
	}
	for l, _ := range lowerCaseL {
		log.Printf("lib: %s", l)
	}
	for l, _ := range targetLs {
		log.Printf("target: %s", l)
	}

}
