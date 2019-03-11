package cc4ce_extensions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path"
	"strings"

	"github.com/pseyfert/compilecommands_to_compilerexplorer/cc2ce"
	"github.com/pseyfert/compilecommands_to_compilerexplorer/cc2ce4lhcb"
)

type CompilerConfig struct {
	Exe      string
	Name     string
	ConfName string
	Options  string
}

func CompilerAndOptions(p cc2ce4lhcb.Project, nightlyroot, cmtconfig string) (CompilerConfig, error) {
	retval, err := CompilerAndOptionsFromJsonByFilename(cc2ce4lhcb.Installarea(p))
	retval.Name = p.Project
	retval.ConfName = p.CE_config_name()
	return retval, err
}

func CompilerAndOptionsFromJsonByFilename(inFileName string) (CompilerConfig, error) {
	var retval CompilerConfig

	db, err := cc2ce.JsonTUsByFilename(inFileName)
	if err != nil {
		return retval, err
	}

	retval.Exe, err = CompilerFromJsonByDB(db)
	retval.Options, err = cc2ce.OptionsFromJsonByDB(db)
	return retval, err
}

func CompilerFromJsonByBytes(inFileContent []byte) (string, error) {
	var db []cc2ce.JsonTranslationunit
	json.Unmarshal(inFileContent, &db)
	return CompilerFromJsonByDB(db)
}

func CompilerFromJsonByDB(db []cc2ce.JsonTranslationunit) (string, error) {
	var b bytes.Buffer
	for _, tu := range db {
		words := strings.Fields(tu.Command)
		for i, w := range words {
			if strings.HasPrefix(w, "-") || strings.HasSuffix(w, ".cpp") {
				break
			}
			if i != 0 {
				b.WriteString(" ")
			}
			b.WriteString(w)
		}
		return b.String(), nil
	}
	return "", fmt.Errorf("no translation units found")
}

func CompilerLibsRpath(bin string) (string, error) {
	log.Printf("trying to determine libstdc++.so from compiler %s", bin)
	output, err := exec.Command(bin, "-print-file-name=libstdc++.so").Output()
	if nil != err {
		return "", fmt.Errorf("call to %s failed: %v", bin, err)
	}
	if len(output) < 2 {
		return "", fmt.Errorf("Received surprisingly short output from %s -print-file-name=libstdc++.so", bin)
	}

	var b bytes.Buffer
	b.WriteString("-Wl,-rpath=")
	b.WriteString(path.Dir(string(output[0 : len(output)-1])))
	return b.String(), nil
}
