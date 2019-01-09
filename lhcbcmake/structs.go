package lhcbcmake

import "github.com/pseyfert/compilecommands_to_compilerexplorer/cc2ce4lhcb"

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
