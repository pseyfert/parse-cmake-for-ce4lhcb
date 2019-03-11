package main

import (
	"bytes"
	"flag"
	"log"
	"os"

	"github.com/pseyfert/compilecommands_to_compilerexplorer/cc2ce4lhcb"
	"github.com/pseyfert/parse-cmake-for-ce4lhcb/cc4ce_extensions"
	"github.com/pseyfert/parse-cmake-for-ce4lhcb/lhcbcmake"
)

func main() {
	// helper
	var p cc2ce4lhcb.Project
	flag.StringVar(&p.Slot, "slot", "lhcb-head", "nightlies slot (i.e. directory in /cvmfs/lhcbdev.cern.ch/nightlies/)")
	flag.StringVar(&p.Day, "day", "Today", "day/buildID (i.e. subdirectory, such as 'Today', 'Mon', or '2032')")
	flag.StringVar(&p.Project, "project", "Rec", "project (such as Rec, Brunel, LHCb, Lbcom)")
	flag.StringVar(&p.Version, "version", "HEAD", "version (i.e. the stuff after the underscore like HEAD or 2016-patches)")
	flag.StringVar(&cc2ce4lhcb.Cmtconfig, "cmtconfig", "x86_64+avx2+fma-centos7-gcc8-opt", "platform, like x86_64+avx2+fma-centos7-gcc7-opt or x86_64-centos7-gcc7-opt")
	flag.StringVar(&cc2ce4lhcb.Nightlyroot, "nightly-base", "/cvmfs/lhcbdev.cern.ch/nightlies/", "add the specified directory to the nightly builds search path")
	flag.Parse()

	cmakeconfig := lhcbcmake.ParseProjectConfigWithDeps(p)

	compilerconf, err := cc4ce_extensions.CompilerAndOptions(p, cc2ce4lhcb.Nightlyroot, cc2ce4lhcb.Cmtconfig)
	if nil != err {
		log.Print("PANIC")
		os.Exit(888)
	}

	compilerconf.Options += PrefixedSeparatorSeparateMap(cmakeconfig.UpperCaseL, "-L", " ")
	compilerconf.Options += " "
	compilerconf.Options += PrefixedSeparatorSeparateMap(cmakeconfig.UpperCaseL, "-Wl,-rpath=", " ")
	compilerconf.Options += " "
	compilerconf.Options += PrefixedSeparatorSeparateMap(cmakeconfig.LowerCaseL, "-l", " ")
	compilerconf.Options += " "
	stdlib, err := cc4ce_extensions.CompilerLibsRpath(compilerconf.Exe)
	if nil != err {
		log.Printf("could not locate standard library location: %v", err)
		log.Printf("trying to cope without ...")
	}
	compilerconf.Options += stdlib
	err = lhcbcmake.WriteConfig([]cc4ce_extensions.CompilerConfig{compilerconf})
	if nil != err {
		log.Printf("something failed: %v", err)
	}
}

func PrefixedSeparatorSeparateMap(stringset map[string]bool, pref, sep string) string {
	var b bytes.Buffer
	addseparator := false
	for k, _ := range stringset {
		if addseparator {
			b.WriteString(sep)
		} else {
			addseparator = true
		}
		b.WriteString(pref)
		b.WriteString(k)
	}
	return b.String()
}
