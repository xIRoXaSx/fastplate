package interpreter

import (
	"bytes"
	"sync"

	"github.com/xiroxasx/fastplate/internal/common"
)

type state struct {
	ignoreIndex        ignoreIndexes
	depsResolver       dependencyResolver
	foreachIndex       foreachIndexes
	varRegistryLocal   variableRegistry
	varRegistryGlobal  variableRegistry // TODO: Currently merging unscopedVarIndexes into this as well!
	varRegistryForeach variableRegistry
	buf                *bytes.Buffer
	*sync.Mutex
}

type ignoreIndexes map[string]ignoreState

type foreachIndexes map[string]int

type dependencies map[string][]string

type variableRegistry struct {
	entries map[string]vars
	*sync.Mutex
}

type vars []common.Variable

type ignoreState uint8

const (
	ignoreStateClose ignoreState = iota
	ignoreStateOpen

	variableRegistryGlobalRegisterGlobal = "global"
)

func (s *state) setGlobalVar(newVar common.Variable) {
	setRegistryVar(&s.varRegistryGlobal, variableRegistryGlobalRegisterGlobal, newVar)
}

func (s *state) setLocalVar(register string, newVar common.Variable) {
	setRegistryVar(&s.varRegistryLocal, register, newVar)
}

func (s *state) setForeachVar(register string, newVar common.Variable) {
	setRegistryVar(&s.varRegistryForeach, register, newVar)
}

func setRegistryVar(reg *variableRegistry, register string, newVar common.Variable) {
	reg.Lock()
	defer reg.Unlock()

	for register, vars := range reg.entries {
		for i, v := range vars {
			if newVar.Name() == v.Name() {
				// Update existing variable.
				reg.entries[register][i] = common.NewVar(v.Name(), newVar.Value())
				return
			}
		}
	}

	reg.entries[register] = append(reg.entries[register], newVar)
}

func (s *state) varLookup(file, name string) (v common.Variable) {
	v = s.varLookupLocal(file, name)
	if v.Name() == "" {
		v = s.varLookupGlobal(name)
	}
	return
}

func (s *state) varLookupGlobal(name string) (v common.Variable) {
	return varLookupRegistry(&s.varRegistryGlobal, variableRegistryGlobalRegisterGlobal, name)
}

func (s *state) varLookupLocal(register, name string) (v common.Variable) {
	return varLookupRegistry(&s.varRegistryLocal, register, name)
}

func (s *state) varLookupForeach(register, name string) (v common.Variable) {
	return varLookupRegistry(&s.varRegistryForeach, register, name)
}

func varLookupRegistry(reg *variableRegistry, register, varName string) (v common.Variable) {
	reg.Lock()
	defer reg.Unlock()

	for _, v := range reg.entries[register] {
		if v.Name() == varName {
			return v
		}
	}

	// If variable is not found, return an empty one.
	return variable{}
}
