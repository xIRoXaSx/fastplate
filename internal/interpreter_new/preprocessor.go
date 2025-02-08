package interpreter

import (
	"bytes"
	"fmt"
)

const (
	directiveNameForeach    = "foreach"
	directiveNameForeachEnd = "foreachend"
	directiveNameIgnore     = "ignore"
	directiveNameIgnoreEnd  = "ignoreend"
	directiveNameImport     = "import"
	directiveNameVariable   = "var"
)

type preprocessorDirective struct {
	name     string
	fileName string
	args     [][]byte
	indent   []byte
	buf      *bytes.Buffer
}

func (i *Interpreter) preprocess(pd *preprocessorDirective, lineDisplayNum int) (err error) {
	callID := fmt.Sprintf("%s: %d", pd.name, lineDisplayNum)
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s: %v", callID, err)
			return
		}
	}()

	switch pd.name {
	case directiveNameForeach:
		// TODO: Implementation.
		return

	case directiveNameForeachEnd:
		// TODO: Implementation.
		return

	case directiveNameIgnore:
		return i.ignoreStart(pd)

	case directiveNameIgnoreEnd:
		return i.ignoreEnd(pd)

	case directiveNameImport:
		return i.importPath(pd)

	case directiveNameVariable:
	default:
		return fmt.Errorf("unknown preprocessor directive: %s", pd.name)
	}

	return
}
