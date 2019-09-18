package lhcbcmake

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pseyfert/compilecommands_to_compilerexplorer/cc2ce4lhcb"
	"github.com/pseyfert/parse-cmake-for-ce4lhcb/cmake"
)

func GaudiProjectDependencies(p cc2ce4lhcb.Project) ([]cc2ce4lhcb.Project, error) {
	var retval = make([]cc2ce4lhcb.Project, 0)
	configpath := filepath.Join(cc2ce4lhcb.Installarea(p), p.Project+"Config.cmake")
	log.Printf("INFO: going into project %s", p.Project)
	log.Printf("INFO: from file %s", configpath)
	// contains the line: set(LHCb_USES Gaudi;master)
	// or (multi deps example): set(DaVinci_USES Analysis;HEAD;Stripping;HEAD)
	projectconfig, err := cmake.Parse(configpath)
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

func ParseProjectConfigWithDeps(p cc2ce4lhcb.Project) ProjectConfig {
	cmakeconfig := ParseProjectConfig(p)

	mapappend := func(a, b map[string]bool) map[string]bool {
		retval := a
		for k, v := range b {
			retval[k] = v
		}
		return retval
	}

	var dependent_projects = map[*cc2ce4lhcb.Project]*ProjectConfig{}
	for _, project := range cmakeconfig.DependsOn {
		dependent_projects[&project] = nil
	}
	needed_external_targets := cmakeconfig.ExternalTargetLs
	resoved_external_targets := cmakeconfig.ExportedTargetLs

	nonil := func(m map[*cc2ce4lhcb.Project]*ProjectConfig) bool {
		all_are_true := true
		for _, v := range m {
			all_are_true = (v != nil)
		}
		return all_are_true
	}

	for !nonil(dependent_projects) {
		for pp, v := range dependent_projects {
			if v != nil {
				continue
			}

			loccmakeconfig := ParseProjectConfig(*pp)
			resoved_external_targets = mapappend(resoved_external_targets, loccmakeconfig.ExportedTargetLs)
			needed_external_targets = mapappend(needed_external_targets, loccmakeconfig.ExternalTargetLs)

			dependent_projects[pp] = &loccmakeconfig
		deploop:
			for _, project := range loccmakeconfig.DependsOn {
				for ppp, _ := range dependent_projects {
					if (*ppp).Project == project.Project {
						continue deploop
					}
				}
				dependent_projects[&project] = nil
			}
		}
	}

	for k, _ := range needed_external_targets {
		if _, found := resoved_external_targets[k]; found {
			delete(needed_external_targets, k)
		}
	}
	for l, _ := range needed_external_targets {
		log.Printf("external targets not found anywhere: %s", l)
	}

	for _, v := range dependent_projects {
		cmakeconfig.UpperCaseL = mapappend(cmakeconfig.UpperCaseL, v.UpperCaseL)
		cmakeconfig.LowerCaseL = mapappend(cmakeconfig.LowerCaseL, v.LowerCaseL)
	}

	return cmakeconfig
}

func ParseProjectConfig(p cc2ce4lhcb.Project) ProjectConfig {
	project := NewProjectConfig()

	var err error
	project.DependsOn, err = GaudiProjectDependencies(p)
	if err != nil {
		log.Print("Error during dependency resolution, continuing with what will work out")
	}

	platformconfigpath := filepath.Join(cc2ce4lhcb.Installarea(p), "cmake", p.Project+"PlatformConfig.cmake")
	thislibdir := filepath.Join(cc2ce4lhcb.Installarea(p), "lib")
	project.UpperCaseL[thislibdir] = true
	platformconfig, err := cmake.Parse(platformconfigpath)
	if err != nil {
		log.Printf("Couldn't open project config: %v", err)
		os.Exit(7)
	}
	for _, funccall := range platformconfig.Functions {
		if funccall.FunctionName == "set(" && funccall.Fargs[0] == p.Project+"_LINKER_LIBRARIES" {
			if len(funccall.Fargs) < 2 {
				log.Printf("ERROR: unexpected line in %s", platformconfigpath)
				log.Printf("       at %v", funccall.Pos)
				log.Printf("       function: %s", funccall.FunctionName)
				log.Printf("       args: %v", funccall.Fargs)
				log.Printf("Project does not seem to declare any library to link against.")
				log.Printf("(This is normal for some projects (Lbcom). Continuing.")
			} else {
				for _, linklib := range strings.Split(funccall.Fargs[1], ";") {
					project.ExportedTargetLs[linklib] = false
				}
			}
		}
	}

	exportfiles, err := filepath.Glob(filepath.Join(cc2ce4lhcb.Installarea(p), "cmake", "*Export.cmake"))
	if err != nil {
		log.Printf("couldn't glob export files: %v", err)
		os.Exit(9)
	}
	for _, exportfile := range exportfiles {
		cmakelists, err := cmake.Parse(exportfile)
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
				if len(loclibs) > 1 {
					log.Printf("WARNING: more than one library in set_target_properties call. Expect undefined behaviour: %v in %s", loclibs, exportfile)
				}
				need_this := false
				for _, loclib := range loclibs {
					if _, found := project.ExportedTargetLs[loclib]; found {
						project.ExportedTargetLs[loclib] = true
						need_this = true
					}
				}
				if !need_this {
					continue
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
							if !strings.HasPrefix(l, "/usr/lib") {
								project.AllLibs[l] = true
							}
						}
					}
					if property == "IMPORTED_SONAME" {
						if strings.HasPrefix(value, "lib") && strings.HasSuffix(value, ".so") {
							project.LowerCaseL[value[3:len(value)-3]] = true
							if loclibs[0] != value[3:len(value)-3] {
								log.Printf("target name and library file name differ: %s -> %s", loclibs[0], value[3:len(value)-3])
							}
						}
					}
				}
			}
		}
	}
	for l, r := range project.ExportedTargetLs {
		if !r {
			log.Printf("WARNING: couldn't resolve link deps for %s", l)
		} else {
			project.LowerCaseL[l] = true
		}
	}
	for l, _ := range project.AllLibs {
		if _, err := os.Stat(l); !os.IsNotExist(err) {
			dir, file := filepath.Split(l)
			validateme := file[3 : len(file)-3]
			if "lib"+validateme+".so" != file {
				log.Print("PANIC")
				os.Exit(777)
			}
			project.UpperCaseL[dir] = true
			project.LowerCaseL[validateme] = true
		} else if strings.HasPrefix(l, "-l") {
			project.LowerCaseL[l[2:len(l)]] = true
		} else {
			if _, found := project.ExportedTargetLs[l]; !found {
				project.TargetLs[l] = true
			}
		}
	}
	for l, _ := range project.TargetLs {
		if _, found := target_to_l[l]; found {
			project.LowerCaseL[l] = true
			delete(project.TargetLs, l)
		}
	}

	for l, _ := range project.UpperCaseL {
		log.Printf("dir: %s", l)
	}
	for l, _ := range project.LowerCaseL {
		log.Printf("lib: %s", l)
	}
	for l, _ := range project.TargetLs {
		log.Printf("target: %s", l)
	}
	for l, _ := range project.ExportedTargetLs {
		log.Printf("linkerlib: %s", l)
	}
	for l, _ := range project.TargetLs {
		if _, found := project.ExportedTargetLs[l]; !found {
			project.ExternalTargetLs[l] = true
			log.Printf("external target: %s", l)
		}
	}
	return project
}
