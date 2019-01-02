package main

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/pseyfert/compilecommands_to_compilerexplorer/cc2ce4lhcb"
)

func NewProjectConfig() ProjectConfig {
	var project ProjectConfig
	project.UpperCaseL = map[string]bool{}
	project.AllLibs = map[string]bool{}
	project.LowerCaseL = map[string]bool{}
	project.TargetLs = map[string]bool{}
	project.ExternalTargetLs = map[string]bool{}
	project.ExportedTargetLs = map[string]bool{}

	return project
}

type ProjectConfig struct {
	// collection of all REQUIRED_LIBRARIES, i.e. transitive link dependencies
	// may be targets, -l calls, paths
	AllLibs          map[string]bool
	UpperCaseL       map[string]bool
	LowerCaseL       map[string]bool
	TargetLs         map[string]bool
	ExternalTargetLs map[string]bool

	// bool: true/false depending if their target property settings have been found
	ExportedTargetLs map[string]bool
	DependsOn        []cc2ce4lhcb.Project
}

func GaudiProjectDependencies(p cc2ce4lhcb.Project) ([]cc2ce4lhcb.Project, error) {
	var retval = make([]cc2ce4lhcb.Project, 0)
	configpath := filepath.Join(cc2ce4lhcb.Installarea(p), p.Project+"Config.cmake")
	// contains the line: set(LHCb_USES Gaudi;master)
	// or (multi deps example): set(DaVinci_USES Analysis;HEAD;Stripping;HEAD)
	projectconfig, err := parse(configpath)
	if err != nil {
		log.Print("ERROR: Couldn't open project config: %v", err)
		return retval, err
	}
	for _, funccall := range projectconfig.Functions {
		if funccall.FunctionName == "set(" && funccall.Fargs[0] == p.Project+"_USES" {
			if len(funccall.Fargs) < 2 {
				log.Printf("WARNING: no 'USES' (i.e. project dependencies) declared")
				log.Printf("         in project %s", p.Project)
				log.Printf("         in file %s:%v", configpath, funccall.Pos)
				log.Printf("       function: %s", funccall.FunctionName)
				log.Printf("       args: %v", funccall.Fargs)
			} else {
				deps := strings.Split(funccall.Fargs[1], ";")
				if len(deps)%2 == 1 {
					log.Printf("ERROR: unexpected USES pattern")
					log.Printf("       has even number of ;")
					log.Printf("       instead found: %s", funccall.Fargs[1])
				}
				for i := 0; i < len(deps); i += 2 {
					var pp cc2ce4lhcb.Project
					pp.Slot = p.Slot
					pp.Day = p.Day
					p.Project = deps[i]
					p.Version = deps[i+1]

					retval = append(retval, p)
				}
			}

		}
	}
	return retval, nil
}
