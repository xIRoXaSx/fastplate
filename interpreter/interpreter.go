package importer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const varFileName = "fastplate.var"

type Interpreter struct {
	opts       *Options
	prefixes   []string
	state      state
	lineEnding []byte
}

type Options struct {
	InPath       string
	OutPath      string
	VarFilePaths VarFilePaths
	Indent       bool
	UseCRLF      bool
	NoStats      bool
}

type VarFilePaths []string

func (vp *VarFilePaths) String() string {
	return strings.Join(*vp, " ")
}

func (vp *VarFilePaths) Set(v string) (err error) {
	*vp = append(*vp, v)
	return
}

type state struct {
	ignoreIndex  map[string]int8
	scopedVars   map[string][]variable
	dependencies map[string][]string
	unscopedVars []variable
	foreach      map[string]foreach
	dirMode      bool
	*sync.Mutex
}

type variable struct {
	name  string
	value string
}

type foreach struct {
	variables []variable
	lines     [][]byte
}

func defaultImportPrefixes() []string {
	return []string{"#fastplate", "# fastplate"}
}

func New(opts *Options) (i Interpreter) {
	i = Interpreter{
		opts:       opts,
		prefixes:   defaultImportPrefixes(),
		lineEnding: []byte("\n"),
		state: state{
			ignoreIndex:  map[string]int8{},
			scopedVars:   map[string][]variable{},
			dependencies: map[string][]string{},
			foreach:      map[string]foreach{},
			Mutex:        &sync.Mutex{},
		},
	}

	if opts.UseCRLF {
		i.lineEnding = []byte("\r\n")
	}

	// Look in the current working directory.
	vFiles := opts.VarFilePaths
	if len(vFiles) == 0 {
		_, err := os.Stat(varFileName)
		if err == nil {
			vFiles = []string{varFileName}
		}
	}

	// Check if the global var files exist and read it into the memory.
	for _, vf := range vFiles {
		_, err := os.Stat(vf)
		if err != nil {
			return
		}
		cont, err := os.ReadFile(vf)
		if err != nil {
			log.Fatal().Err(err).Str("path", vf).Msg("unable to read global variable file")
		}
		lines := bytes.Split(cont, i.lineEnding)
		for _, l := range lines {
			split := bytes.Split(i.CutPrefix(l), []byte{' '})
			if string(split[0]) != commandVar {
				continue
			}
			// Skip the var declaration keyword.
			i.setUnscopedVar(split[1:])
		}
	}

	return
}

func (i *Interpreter) TrimLine(b, prefix []byte) []byte {
	return bytes.Trim(bytes.TrimPrefix(b, prefix), string(i.lineEnding)+" ")
}

func (i *Interpreter) CutPrefix(b []byte) (ret []byte) {
	prefix := i.matchedImportPrefix(b)
	if prefix == nil {
		return
	}
	return i.TrimLine(b, prefix)
}

func (i *Interpreter) Start() (err error) {
	stat, err := os.Stat(i.opts.InPath)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to stat input path")
	}

	start := time.Now()
	i.state.dirMode = stat.IsDir()
	if i.state.dirMode {
		err = i.runDirMode()
	} else {
		err = i.runFileMode()
	}
	el := time.Since(start)
	if err != nil {
		return
	}

	if !i.opts.NoStats {
		fmt.Println("Execution took", el)
	}
	return
}

// runDirMode runs the import for each file inside the Options.InPath.
func (i *Interpreter) runDirMode() (err error) {
	const dirPerm = os.FileMode(0700)

	err = os.MkdirAll(i.opts.OutPath, dirPerm)
	if err != nil {
		return
	}

	err = filepath.WalkDir(i.opts.InPath, func(inPath string, d os.DirEntry, err error) error {
		dest := strings.ReplaceAll(inPath, i.opts.InPath, i.opts.OutPath)
		if d.IsDir() {
			if dest == "" {
				return nil
			}
			err = os.MkdirAll(dest, dirPerm)
			if err != nil {
				return err
			}
			return nil
		}

		// Write to the buffer to ensure that files don't get partially written.
		buf := &bytes.Buffer{}
		err = i.interpretFile(inPath, nil, buf)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
		if err != nil {
			return err
		}
		defer func() {
			cErr := out.Close()
			if err == nil {
				err = cErr
			}
		}()

		// Write buffer to the file and cut last new line.
		_, err = out.Write(buf.Bytes()[:buf.Len()-1])
		return err
	})
	return
}

// runFileMode runs the import with the targeted Options.OutPath.
func (i *Interpreter) runFileMode() (err error) {
	out, err := os.OpenFile(i.opts.OutPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	defer func() {
		cErr := out.Close()
		if err == nil {
			err = cErr
		}
	}()

	// Write to the buffer to ensure that files don't get partially written.
	buf := &bytes.Buffer{}
	err = i.interpretFile(i.opts.InPath, nil, buf)
	if err != nil {
		return
	}

	// Write buffer to the file and cut last new line.
	_, err = out.Write(buf.Bytes()[:buf.Len()-1])
	return
}

func (i *Interpreter) interpretFile(filePath string, indent []byte, out io.Writer) (err error) {
	cont, err := os.ReadFile(filePath)
	if err != nil {
		log.Warn().Err(err).Str("file", filePath).Msg("unable to read file")
		return
	}

	// Append indention to all linebreaks, prepend to the first line.
	cutSet := i.lineEnding
	if len(indent) > 0 {
		cont = bytes.ReplaceAll(cont, cutSet, append(cutSet, indent...))
		cont = append(indent, cont...)
	}

	lines := bytes.Split(cont, cutSet)
	for _, l := range lines {
		if i.opts.Indent {
			indent = leadingIndents(l)
		}

		// Skip the indents.
		linePart := l[len(indent):]
		prefix := i.matchedImportPrefix(linePart)
		if prefix == nil {
			// Line does not contain one of the required prefixes.
			if i.state.ignoreIndex[filePath] == 1 {
				// Still in an ignore block.
				continue
			}
			if len(i.state.foreach[filePath].variables) > 0 {
				// Currently moving inside a foreach loop.
				i.appendLine(filePath, l)
				continue
			}

			var ret []byte
			ret, err = i.resolve(filePath, l)
			if err != nil {
				return
			}
			_, err = out.Write(append(ret, cutSet...))
			if err != nil {
				return
			}
		} else {
			// Trim statement and check against internal commands.
			statement := i.TrimLine(linePart, prefix)
			split := bytes.Split(statement, []byte{' '})
			if len(split) > 0 && string(split[0]) != commandImport {
				err = i.executeCommand(string(split[0]), filePath, split[1:], out)
				if err != nil {
					return
				}
				continue
			}

			if len(split) < 2 {
				err = errors.New("no import path given")
				return
			}
			stmnt := filepath.Clean(string(split[1]))
			filePath = filepath.Clean(filePath)
			if i.state.hasCyclicDependency(filePath, stmnt) {
				err = fmt.Errorf("detected import cycle: %s -> %s", filePath, stmnt)
				return
			}
			i.state.addDependency(filePath, stmnt)
			err = i.interpretFile(stmnt, indent, out)
			if err != nil {
				return err
			}
		}
	}
	return
}

func (i *Interpreter) matchedImportPrefix(line []byte) []byte {
	for _, pref := range i.prefixes {
		if bytes.HasPrefix(line, []byte(pref)) {
			return []byte(pref)
		}
	}
	return nil
}

func leadingIndents(line []byte) (s []byte) {
	for _, r := range line {
		if r != ' ' && r != '\t' {
			break
		}
		s = append(s, r)
	}
	return
}
