package interpreter

import (
	"bytes"
	"errors"
)

var (
	errEmptyVariableParameter = errors.New("variable name or value must not be empty")
)

type variable struct {
	name  string
	value string
}

// Implements the [common.Variable] interface.
func (v variable) Name() string {
	return v.name
}

// Implements the [common.Variable] interface.
func (v variable) Value() string {
	return v.value
}

// setLocalVar parses and sets a local variable from the given args.
func (i *Interpreter) setLocalVar(scope string, args [][]byte) (err error) {
	i.state.varRegistryLocal.Lock()
	defer i.state.varRegistryLocal.Unlock()

	v := variableFromArgs(args)
	if v.Name() == "" || v.Value() == "" {
		return errEmptyVariableParameter
	}

	for index, localVar := range i.state.varRegistryLocal.entries[scope] {
		if localVar.Name() == v.Name() {
			i.state.varRegistryLocal.entries[scope][index] = v
			return
		}
	}

	i.state.varRegistryLocal.entries[scope] = append(i.state.varRegistryLocal.entries[scope], v)
	return
}

// variableFromArgs parses [variable] from the given args.
func variableFromArgs(args [][]byte) (v variable) {
	if len(args) == 0 {
		return
	}

	for idx, token := range args {
		// Skip empty tokens.
		if len(token) == 0 {
			continue
		}

		// Syntax: "x = y".
		if bytes.Equal(token, []byte{'='}) {
			varNameIdx := 0
			if idx > 0 {
				varNameIdx = idx - 1
			}
			var remainder []byte
			if idx < len(args)-1 {
				// Skip the equal sign.
				remainder = bytes.Join(args[idx+1:], []byte{})
			}
			return variable{
				name:  string(bytes.Join(args[:varNameIdx], []byte{})),
				value: string(remainder),
			}
		}

		tok := bytes.SplitN(token, []byte{'='}, 2)
		if len(tok) == 2 {
			name := tok[0]
			value := tok[1]

			// Syntax: "x= y".
			if len(value) == 0 {
				return variable{
					name:  string(tok[0]),
					value: string(bytes.Join(args[idx+1:], []byte{})),
				}
			}

			// Syntax: "x =y".
			if len(name) == 0 {
				return variable{
					name:  string(bytes.Join(args[:idx], []byte{})),
					value: string(value),
				}
			}

			// Syntax: "x=y".
			return variable{
				name:  string(name),
				value: string(value),
			}
		}
	}

	return
}
