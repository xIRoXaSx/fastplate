package interpreter

import (
	"fmt"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	r "github.com/stretchr/testify/require"
)

func TestVariableFromArgs(t *testing.T) {
	t.Parallel()

	type testCase struct {
		content       [][]byte
		expectedName  string
		expectedValue string
	}

	const (
		keyName  = "name"
		keyValue = "value"
	)

	keyValuePair := func(k, v string) []byte {
		return []byte(fmt.Sprintf("%s=%s", k, v))
	}

	testCases := []testCase{
		{content: nil},
		{content: [][]byte{[]byte("")}},
		{content: [][]byte{[]byte("=")}},
		{
			expectedValue: keyValue,
			content:       [][]byte{keyValuePair("", keyValue)},
		},
		{
			expectedValue: "=",
			content:       [][]byte{keyValuePair("", "=")},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content:       [][]byte{keyValuePair(keyName, keyValue)},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte("="),
				[]byte(""),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte(""),
				[]byte(""),
				[]byte("="),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte(""),
				[]byte("="),
				[]byte(""),
				[]byte(""),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte("="),
				[]byte(""),
				[]byte(""),
				[]byte(""),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName + "="),
				[]byte(""),
				[]byte(""),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte(""),
				[]byte(""),
				[]byte("=" + keyValue),
			},
		},
	}

	for i, tc := range testCases {
		v := variableFromArgs(tc.content)
		r.Exactly(t, tc.expectedName, v.name, "names do not match: case=%d, expected=%s, actual=%s", i, tc.expectedName, v.name)
		r.Exactly(t, tc.expectedValue, v.value, "names do not match: case=%d, expected=%s, actual=%s", i, tc.expectedValue, v.value)
	}
}

func TestSetLocalVar(t *testing.T) {
	t.Parallel()

	const (
		keyRegisterName = "testName"
		keyName         = "name"
		keyValue        = "value"
		keyNameNew      = "newName"
		keyValueNew     = "newValue"
	)

	i := New(&Options{NoStats: true}, zerolog.New(os.Stderr))
	err := i.setLocalVar(keyRegisterName, [][]byte{[]byte(keyName + "=" + keyValue)})
	r.NoError(t, err)
	localVars := i.state.varRegistryLocal.entries[keyRegisterName]
	r.Len(t, localVars, 1)
	v := localVars[0]
	r.Exactly(t, keyName, v.Name())
	r.Exactly(t, keyValue, v.Value())

	// Variables with the equal names should be updated.
	err = i.setLocalVar(keyRegisterName, [][]byte{[]byte(keyName + "=" + keyValueNew)})
	r.NoError(t, err)
	localVars = i.state.varRegistryLocal.entries[keyRegisterName]
	r.Len(t, localVars, 1)
	v = localVars[0]
	r.Exactly(t, keyName, v.Name())
	r.Exactly(t, keyValueNew, v.Value())

	// New variables with the same value should be added.
	err = i.setLocalVar(keyRegisterName, [][]byte{[]byte(keyNameNew + "=" + keyValueNew)})
	r.NoError(t, err)
	localVars = i.state.varRegistryLocal.entries[keyRegisterName]
	r.Len(t, localVars, 2)
	v = localVars[1]
	r.Exactly(t, keyNameNew, v.Name())
	r.Exactly(t, keyValueNew, v.Value())

	// Empty variable names and values should return an error.
	err = i.setLocalVar(keyRegisterName, [][]byte{[]byte("=" + keyValueNew)})
	r.ErrorIs(t, err, errEmptyVariableParameter)
	err = i.setLocalVar(keyRegisterName, [][]byte{[]byte(keyName + "=")})
	r.ErrorIs(t, err, errEmptyVariableParameter)
}

func TestVariableSetAndGet(t *testing.T) {
	t.Parallel()

	type testCase struct {
		content       [][]byte
		expectedName  string
		expectedValue string
	}

	const (
		keyName  = "name"
		keyValue = "value"
	)

	l := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ip := New(&Options{NoStats: true}, l)

	testCases := []testCase{
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte("="),
				[]byte(""),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte(""),
				[]byte(""),
				[]byte("="),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte(""),
				[]byte("="),
				[]byte(""),
				[]byte(""),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte("="),
				[]byte(""),
				[]byte(""),
				[]byte(""),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName + "="),
				[]byte(""),
				[]byte(""),
				[]byte(keyValue),
			},
		},
		{
			expectedName:  keyName,
			expectedValue: keyValue,
			content: [][]byte{
				[]byte(keyName),
				[]byte(""),
				[]byte(""),
				[]byte(""),
				[]byte("=" + keyValue),
			},
		},
	}

	for i, tc := range testCases {
		v := variableFromArgs(tc.content)

		// Global variables.
		// Set a new variable.
		ip.state.setGlobalVar(v)
		rvar := ip.state.varLookupGlobal(v.name)
		r.Exactly(t, v.name, rvar.Name(), "global var name differs: case=%d, expected=%s, actual=%s", i, v.name, rvar.Name())
		r.Exactly(t, v.value, rvar.Value(), "global var value differs: case=%d, expected=%s, actual=%s", i, v.value, rvar.Value())
		// Unset the variable's value.
		ip.state.setGlobalVar(variable{name: v.name})
		rvar = ip.state.varLookupGlobal(v.name)
		r.Exactly(t, v.name, rvar.Name(), "cleared global var name differs: case=%d, expected=%s, actual=%s", i, v.name, rvar.Name())
		r.Exactly(t, "", rvar.Value(), "cleared global var value differs: case=%d, expected=%s, actual=%s", i, v.value, rvar.Value())

		// Local variables.
		// Set a new variable.
		register := "test123"
		ip.state.setLocalVar(register, v)
		rvar = ip.state.varLookupLocal(register, v.name)
		r.Exactly(t, v.name, rvar.Name(), "local var name differs: case=%d, expected=%s, actual=%s", i, v.name, rvar.Name())
		r.Exactly(t, v.value, rvar.Value(), "local var value differs: case=%d, expected=%s, actual=%s", i, v.value, rvar.Value())
		// Unset the variable's value.
		ip.state.setLocalVar(register, variable{name: v.name})
		rvar = ip.state.varLookupLocal(register, v.name)
		r.Exactly(t, v.name, rvar.Name(), "cleared local var name differs: case=%d, expected=%s, actual=%s", i, v.name, rvar.Name())
		r.Exactly(t, "", rvar.Value(), "cleared local var value differs: case=%d, expected=%s, actual=%s", i, v.value, rvar.Value())

		// Foreach variables.
		// Set a new variable.
		register = "testtest123"
		ip.state.setForeachVar(register, v)
		rvar = ip.state.varLookupForeach(register, v.name)
		r.Exactly(t, v.name, rvar.Name(), "local var name differs: case=%d, expected=%s, actual=%s", i, v.name, rvar.Name())
		r.Exactly(t, v.value, rvar.Value(), "local var value differs: case=%d, expected=%s, actual=%s", i, v.value, rvar.Value())
		// Unset the variable's value.
		ip.state.setForeachVar(register, variable{name: v.name})
		rvar = ip.state.varLookupForeach(register, v.name)
		r.Exactly(t, v.name, rvar.Name(), "cleared local var name differs: case=%d, expected=%s, actual=%s", i, v.name, rvar.Name())
		r.Exactly(t, "", rvar.Value(), "cleared local var value differs: case=%d, expected=%s, actual=%s", i, v.value, rvar.Value())

	}
}
