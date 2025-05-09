package functions

import (
	"os"
	"path/filepath"

	"github.com/xiroxasx/yatt/internal/common"
)

func Var(fileName string, args [][]byte, additionalVars []common.Variable, varSetter func(name, value []byte) error) (ret []byte, err error) {
	err = assertArgsLengthAtLeast(args, 2)
	if err != nil {
		return
	}

	// Check if any additional variable matches.
	arg0 := args[0]
	arg1 := args[1]
	for _, v := range additionalVars {
		if string(arg0) == v.Name() {
			arg1 = []byte(v.Value())
			break
		}
	}

	err = varSetter(arg0, arg1)
	return
}

func Env(args [][]byte) (ret []byte, err error) {
	err = assertArgsLengthExact(args, 1)
	if err != nil {
		return
	}

	ret = []byte(os.Getenv(string(args[0])))
	return
}

func FileBaseName(path string) (ret []byte, err error) {
	ret = []byte(filepath.Base(path))
	return
}

func FileName(path string) (ret []byte, err error) {
	ret = []byte(path)
	return
}
