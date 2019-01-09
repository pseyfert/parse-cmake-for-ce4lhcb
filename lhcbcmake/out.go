package lhcbcmake

import (
	"bytes"
	"fmt"
	"log"

	write "github.com/google/renameio"
	"github.com/pseyfert/parse-cmake-for-ce4lhcb/cc4ce_extensions"
)

func WriteConfig(confs []cc4ce_extensions.CompilerConfig) error {
	f, err := write.TempFile("", "./c++.defaults.properties")
	if err != nil {
		log.Printf("Couldn't create tempfile for output writing: %v", err)
		return err
	}
	defer f.Cleanup()

	if _, err := fmt.Fprint(f, "compilers=&autogen\n"); err != nil {
		log.Print("Error writing to config: %v", err)
		return err
	}
	{
		var b bytes.Buffer
		addseparator := false
		for _, c := range confs {
			if addseparator {
				b.WriteString(":")
			} else {
				addseparator = true
			}
			b.WriteString(c.ConfName)
		}
		if _, err := fmt.Fprintf(f, "group.autogen.compilers=%s\n", b.String()); err != nil {
			log.Print("Error writing to config: %v", err)
			return err
		}
	}
	if _, err := fmt.Fprint(f, "group.autogen.groupName=auto-generated compiler settings\n"); err != nil {
		log.Print("Error writing to config: %v", err)
		return err
	}
	compiler_writer := func(c cc4ce_extensions.CompilerConfig) error {
		if _, err := fmt.Fprintf(f, "compiler.%s.name=%s\n", c.ConfName, c.Name); err != nil {
			log.Print("Error writing to config: %v", err)
			return err
		}
		if _, err := fmt.Fprintf(f, "compiler.%s.exe=%s\n", c.ConfName, c.Exe); err != nil {
			log.Print("Error writing to config: %v", err)
			return err
		}
		if _, err := fmt.Fprintf(f, "compiler.%s.options=%s\n", c.ConfName, c.Options); err != nil {
			log.Print("Error writing to config: %v", err)
			return err
		}
		return nil
	}
	for _, c := range confs {
		if err := compiler_writer(c); err != nil {
			return err
		}
	}

	if err := f.CloseAtomicallyReplace(); err != nil {
		log.Printf("writing c++.defaults.properties failed: %v", err)
		return err
	}
	return nil
}
