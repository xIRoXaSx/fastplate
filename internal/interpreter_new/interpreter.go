package interpreter

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	templateStart = "{{"
	templateEnd   = "}}"
	prefixName    = "fastplate"
	varFileName   = "fastplate.var"
)

var (
	templateStartBytes = []byte(templateStart)
	templateEndBytes   = []byte(templateEnd)
)

type Interpreter struct {
	prefixes   []string
	lineEnding []byte
	l          zerolog.Logger
	state      *state

	opts *Options
}

type Options struct {
	InPath        string
	OutPath       string
	FileWhitelist []string
	FileBlacklist []string
	VarFilePaths  []string
	Indent        bool
	NoStats       bool
	Verbose       bool
}

func New(opts *Options, l zerolog.Logger) (i *Interpreter) {
	i = &Interpreter{
		opts:       opts,
		prefixes:   defaultPrefixTokens(),
		lineEnding: []byte(lineEnding),
		l:          l,
		state: &state{
			ignoreIndex:  make(ignoreIndexes, 0),
			buf:          &bytes.Buffer{},
			Mutex:        &sync.Mutex{},
			depsResolver: newDependencyResolver(),
			varRegistryLocal: variableRegistry{
				entries: make(map[string]vars, 0),
				Mutex:   &sync.Mutex{},
			},
			varRegistryGlobal: variableRegistry{
				entries: make(map[string]vars, 0),
				Mutex:   &sync.Mutex{},
			},
			varRegistryForeach: variableRegistry{
				entries: make(map[string]vars, 0),
				Mutex:   &sync.Mutex{},
			},
		},
	}

	i.initScopedVars()
	return
}

func (i *Interpreter) Start() (err error) {
	i.opts.InPath = filepath.Clean(i.opts.InPath)

	stat, err := os.Stat(i.opts.InPath)
	if err != nil {
		i.l.Fatal().Err(err).Msg("unable to stat input path")
	}

	start := time.Now()
	defer func() {
		if err != nil {
			return
		}

		elapsed := time.Since(start)
		if !i.opts.NoStats {
			fmt.Println("Execution took", elapsed)
		}
	}()

	if stat.IsDir() {
		err = i.runDirMode(i.opts.InPath, i.opts.OutPath)
		if err != nil {
			return
		}
	} else {
		outDir := filepath.Clean(filepath.Dir(i.opts.OutPath))
		err = os.MkdirAll(outDir, 0o755)
		if err != nil {
			return
		}
		err = i.runFileMode(i.opts.InPath, i.opts.OutPath)
		if err != nil {
			return
		}
	}

	return
}

func defaultPrefixTokens() []string {
	return []string{
		fmt.Sprintf("#%s", prefixName),
		fmt.Sprintf("# %s", prefixName),
		fmt.Sprintf("//%s", prefixName),
		fmt.Sprintf("// %s", prefixName),
	}
}

// interpret tries to interpret the scanned content of file.rc.
// If the ReadCloser content contains available tokens, it tries to resolve them and writes it,
// along with the prepended indentParent, to buf.
func (i *Interpreter) interpret(file interpreterFile, parentLineIndent []byte) (err error) {
	// Always ensure to close the file's rc.
	defer func() {
		cErr := file.rc.Close()
		if cErr != nil {
			if err == nil {
				err = cErr
				return
			}
			i.l.Err(cErr).Str("file", file.name).Msg("closing file reader")
		}
	}()

	var (
		// Currently read line.
		lineNum int
		// Limits reads to 65536 bytes per line.
		scanner = bufio.NewScanner(file.rc)
	)
	for scanner.Scan() {
		err = scanner.Err()
		if err != nil {
			return
		}

		line := scanner.Bytes()
		currentLineIndent := make([]byte, 0)
		if i.opts.Indent {
			// Line indents are required, check current line indents.
			currentLineIndent = getLeadingWhitespace(line)
		}

		err = i.searchTokensAndExecute(file.name, line, currentLineIndent, parentLineIndent, file.writer, lineNum+1)
		if err != nil {
			return
		}
	}

	return
}

func (i *Interpreter) cutPrefix(b []byte) (ret []byte) {
	prefix := i.matchedPrefixToken(b)
	if prefix == nil {
		return
	}
	return i.trimLine(b, prefix)
}

func (i *Interpreter) trimLine(b, prefix []byte) []byte {
	return bytes.TrimPrefix(bytes.TrimSpace(b), append(prefix, ' '))
}

func (i *Interpreter) matchedPrefixToken(line []byte) (prefix []byte) {
	for _, pref := range i.prefixes {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte(pref)) {
			return []byte(pref)
		}
	}
	return
}

func (i *Interpreter) initScopedVars() {
	// Look for var file in the current working directory.
	vFiles := i.opts.VarFilePaths
	if len(vFiles) == 0 {
		_, err := os.Stat(varFileName)
		if err == nil {
			vFiles = []string{varFileName}
		}
	}

	// Check if the global var files exist and read it into the memory.
	for _, vf := range vFiles {
		cont, err := os.ReadFile(vf)
		if err != nil {
			i.l.Fatal().Err(err).Str("path", vf).Msg("unable to read variable file")
		}

		lines := bytes.Split(cont, i.lineEnding)
		for _, l := range lines {
			split := bytes.Split(i.cutPrefix(l), []byte{' '})
			if string(split[0]) != directiveNameVariable {
				continue
			}
			// Skip the var declaration keyword.
			i.setGlobalVar(split[1:])
		}
	}
}

// setGlobalVar parses and registers an unscoped variable from the given args.
func (i *Interpreter) setGlobalVar(tokens [][]byte) {
	variable := variableFromArgs(tokens)
	i.state.setGlobalVar(variable)
}

func (i *Interpreter) writeInterpretedFile(inPath, outPath string) (err error) {
	out, err := os.OpenFile(outPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	defer func() {
		cErr := out.Close()
		if err == nil {
			err = cErr
		}
	}()

	// Copy file contents if the current file is matching the filters,
	// we don't need to interpret them.
	isRaw, err := i.rawCopyOnListMatch(inPath, out)
	if err != nil {
		return err
	}
	if isRaw {
		return nil
	}

	// Open the input file.
	// The interpret method will close it afterwards.
	inFile, err := os.Open(inPath)
	if err != nil {
		return
	}

	buf := &bytes.Buffer{}
	interFile := interpreterFile{
		name:   inPath,
		rc:     inFile,
		writer: buf,
	}
	// Write to the buffer to ensure that files don't get partially written.
	err = i.interpret(interFile, nil)
	if err != nil {
		return
	}

	if buf.Len() > 0 {
		// Write buffer to the file and cut last new line.
		_, err = out.Write(buf.Bytes()[:buf.Len()-1])
	}
	return
}

func (i *Interpreter) rawCopyOnListMatch(inPath string, out io.Writer) (isRaw bool, err error) {
	writeTo := func(inPath string, out io.Writer) (err error) {
		var b []byte
		b, err = os.ReadFile(inPath)
		if err != nil {
			return
		}

		// Write file content to the output.
		_, err = out.Write(b)
		if err != nil {
			return
		}
		return
	}

	if i.matchedBlacklist(inPath) {
		log.Debug().Str("file", inPath).Msg("matched blacklist, plain copy")
		isRaw = true
		err = writeTo(inPath, out)
		return
	}

	if len(i.opts.FileWhitelist) == 0 || i.matchedWhitelist(inPath) {
		return
	}

	log.Debug().Str("file", inPath).Msg("does not match whitelist, plain copy")
	isRaw = true
	err = writeTo(inPath, out)
	return
}

func (i *Interpreter) matchedBlacklist(v string) (matched bool) {
	if i.opts == nil {
		return
	}

	for _, f := range i.opts.FileBlacklist {
		reg := regexp.MustCompile(f)
		m := reg.MatchString(v)
		if m {
			return true
		}
	}

	return
}

func (i *Interpreter) matchedWhitelist(v string) (matched bool) {
	if i.opts == nil {
		return
	}

	for _, f := range i.opts.FileWhitelist {
		reg := regexp.MustCompile(f)
		m := reg.MatchString(v)
		if m {
			return true
		}
	}

	return
}
