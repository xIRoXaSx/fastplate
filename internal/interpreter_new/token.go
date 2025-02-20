package interpreter

import (
	"bytes"
	"io"
	"path/filepath"

	"github.com/xiroxasx/fastplate/internal/common"
)

type interpreterFile struct {
	name   string
	rc     io.ReadCloser
	writer io.Writer
}

type recurringToken uint8

const (
	recurringTokenIgnore recurringToken = iota + 1
	recurringTokenForeach
)

func (i *Interpreter) preprocessorState(fileName string) (t recurringToken) {
	// Line does not contain one of the required prefixes.
	if i.state.ignoreIndex[fileName] == ignoreStateOpen {
		return recurringTokenIgnore
	} else if i.state.foreachIndex[fileName] != 0 {
		return recurringTokenForeach
	}

	return
}

func (i *Interpreter) searchTokensAndExecute(fileName string, line, currentLineIndent, parentLineIndent []byte, buf io.Writer, lineNum int) (err error) {
	lineDisplayNum := lineNum + 1
	lineNoIndent := line[len(currentLineIndent):]
	prefix := i.matchedPrefixToken(lineNoIndent)
	if len(prefix) > 0 {
		// Trim the prefix and check against internal commands.
		statement := i.trimLine(lineNoIndent, prefix)
		split := bytes.Split(statement, []byte{' '})
		if len(split) == 0 {
			return
		}

		pd := &preprocessorDirective{
			name:     string(split[0]),
			fileName: filepath.Clean(fileName),
			args:     split[1:],
			indent:   append(currentLineIndent, parentLineIndent...),
			buf:      &bytes.Buffer{},
		}
		err = i.preprocess(pd, lineDisplayNum)
		if err != nil {
			return
		}

		_, err = pd.buf.WriteTo(buf)
		return
	}

	switch i.preprocessorState(fileName) {
	case recurringTokenIgnore:

		// Currently moving inside a ignore block, skipping line...
		return

	case recurringTokenForeach:
		// TODO: Implementation.
		return

	default:
		// No prefix found, try to resolve variables and functions if there are any.
		var ret []byte
		ret, err = i.resolve(fileName, line, nil)
		if err != nil {
			return
		}
		indents := append(parentLineIndent, currentLineIndent...)
		ret = append(indents, ret...)
		_, err = buf.Write(append(ret, i.lineEnding...))
		if err != nil {
			return
		}
	}

	return
}

func (i *Interpreter) resolve(fileName string, line []byte, additionalVars []common.Variable) (ret []byte, err error) {
	templateStart := templateStartBytes
	templateEnd := templateEndBytes
	ret = line
	tokens := bytes.Split(line, templateStart)
	if len(tokens) == 1 {
		// Nothing needs to be resolved.
		return
	}

	for _, token := range tokens {
		match := bytes.Split(token, templateEnd)
		if len(match) == 1 {
			continue
		}

		// Resolve functions and variables.
		// If no matched variable is found, try to find an global var.
		for _, m := range match {
			if len(m) == 0 {
				continue
			}

			fncName, args := i.unwrapFunc(m)
			if len(fncName) == 0 {
				// No function found, try to lookup and replace variable.
				v := i.state.varLookup(fileName, string(m))
				if v.Value() == "" {
					continue
				}

				ret = replaceVar(ret, m, []byte(v.Value()))
				continue
			}

			// Check function's args for variables.
			varsFromArgs := make([]common.Variable, 0)
			for j := range args {
				v := i.state.varLookup(fileName, string(args[j]))
				if v.Name() == "" {
					// For some functions, numbers are also used. Add them.
					val := string(args[j])
					v = variable{name: val, value: val}
				}
				varsFromArgs = append(varsFromArgs, v)
			}
			if len(varsFromArgs) == 0 {
				continue
			}

			var remappedArgs [][]byte
			fncNameStr := fncName.string()
			remappedArgs, err = remapArgsWithVariables(fncNameStr, varsFromArgs, varsFromArgs)
			if err != nil {
				return
			}

			var mod []byte
			mod, err = i.executeFunction(fncName, fileName, remappedArgs, additionalVars)
			if err != nil {
				return
			}
			ret = replaceVar(ret, m, mod)
		}
	}

	if len(bytes.Split(ret, templateStart)) > 1 && !bytes.Equal(ret, line) {
		ret, err = i.resolve(fileName, ret, additionalVars)
		if err != nil {
			return
		}
	}

	for _, v := range additionalVars {
		ret = replaceVar(line, []byte(v.Name()), []byte(v.Value()))
	}
	return
}

type interpreterFunc []byte

func (i interpreterFunc) string() string {
	return string(i)
}

// unwrapFunc gets the function's name and its args from the given byte slice.
func (i *Interpreter) unwrapFunc(b []byte) (fncName interpreterFunc, args [][]byte) {
	args = make([][]byte, 0)
	fnc := bytes.SplitN(bytes.TrimSpace(b), []byte("("), 2)
	if len(fnc) == 1 {
		return
	}

	fncName = bytes.TrimSpace(fnc[0])
	fnc[1] = bytes.Split(fnc[1], []byte(")"))[0]
	args = bytes.Split(fnc[1], []byte(","))
	for j := range args {
		args[j] = bytes.TrimSpace(args[j])
	}
	return
}

//
// Helper functions.
//

func replaceVar(line, varName, replacement []byte) []byte {
	matched := bytes.Join([][]byte{templateStartBytes, varName, templateEndBytes}, nil)
	return bytes.ReplaceAll(line, matched, replacement)
}

func remapArgsWithVariables(fncNameStr string, varsFromArgs, additionalVars []common.Variable) (values [][]byte, err error) {
	values = make([][]byte, len(varsFromArgs))

additionalVar:
	for idx := range varsFromArgs {
		for _, av := range additionalVars {
			// Overwrite variable value if the names match.
			// This may be the case for "foreach"-variables.
			if varsFromArgs[idx].Name() == av.Name() {
				values[idx] = []byte(av.Value())
				continue additionalVar
			}
		}

		// Keep variable name intact so the function call can retrieve the var's value.
		if fncNameStr == "var" {
			values[idx] = []byte(varsFromArgs[idx].Name())
			continue
		}
		values[idx] = []byte(varsFromArgs[idx].Value())
	}

	return
}

// getLeadingWhitespace returns all leading whitespace characters.
func getLeadingWhitespace(line []byte) (s []byte) {
	for _, r := range line {
		if r != ' ' && r != '\t' {
			return
		}
		s = append(s, r)
	}
	return
}
