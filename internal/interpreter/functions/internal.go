package functions

import (
	"errors"

	"github.com/xiroxasx/fastplate/internal/common"
)

func Var(args [][]byte, additionalVars []common.Var) (ret [][]byte, err error) {
	if len(args) != 2 {
		err = errors.New("exactly 2 args expected")
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
	ret = [][]byte{arg0, arg1}
	return
}
