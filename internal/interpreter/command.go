package interpreter

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
)

const (
	commandIgnore     = "ignore"
	commandVar        = "var"
	commandForeach    = "foreach"
	commandForeachEnd = "foreachend"
	commandImport     = "import"
)

func (i *Interpreter) executeCommand(command, file string, args [][]byte, lineNum int, callID string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s: %v (%s)", command, err, callID)
		}
	}()

	switch command {
	case commandIgnore:
		if len(args) != 1 {
			return errors.New("exactly 1 arg expected, \"start\" or \"end\"")
		}
		i.ignore(file, string(args[0]))

	case commandVar:
		if len(args) < 2 {
			return errors.New("expected a name and a value")
		}
		i.setScopedVar(file, args)

	case commandForeach:
		if len(args) < 1 {
			return errors.New("at least 1 arg expected")
		}

		var fe foreach
		ln := lineNum
		fe, err = i.state.foreachLoad(file)
		if err != nil {
			if err != errMapLoadForeach {
				return
			}

			err = nil
			fe = foreach{
				open: 1,
				buf:  queue{v: []foreachBuffer{{ln: ln}}},
				mx:   &sync.Mutex{},
			}

		} else {
			buf := foreachBuffer{ln: ln}
			bufLen := fe.buf.len()
			if bufLen-1 > 0 {
				lastBuf := fe.buf.last()
				lastBuf.nextRef = append(lastBuf.nextRef, fe.c.p)
			}

			var ref *foreachBuffer
			if fe.c.p >= 0 && len(fe.buf.v) > 0 {
				ref = fe.buf.firstN(fe.c.p)
				ref.startNext = append(ref.startNext, len(ref.lines))
			}

			// Check if line is directly nested.
			if (fe.c.p > 0 && buf.ln-fe.buf.v[bufLen-1].ln == 1) && len(fe.buf.v[bufLen-1].lines) == 0 {
				fe.buf.v[bufLen-1].lines = [][]byte{{}}
			}

			fe.buf.push(buf)
			fe.c.j = bufLen - fe.c.p
			fe.c.p = bufLen
			fe.open++
		}

		i.state.foreach.Store(file, fe)
		for _, arg := range args {
			// Brackets are optional, trim them.
			arg = bytes.Trim(bytes.Trim(bytes.TrimSpace(arg), "["), "]")
			if arg == nil {
				continue
			}

			b := bytes.Split(arg, []byte(","))
			for _, trim := range b {
				if len(trim) == 0 {
					continue
				}

				// Trim braces to get variable name.
				trim = bytes.Trim(bytes.Trim(trim, "{{"), "}}")
				err = i.setForeachVar(file, string(trim))
				if err != nil {
					return
				}
			}
		}

	case commandForeachEnd:
		var fe foreach
		fe, err = i.state.foreachLoad(file)
		if err != nil {
			return
		}

		// Wait until each foreach loop is closed.
		fe.open--
		fe.c.p -= fe.c.j
		i.state.foreach.Store(file, fe)
		if fe.open > 0 {
			return
		}

		err = i.evaluateForeach(fe, file)
		if err != nil {
			return
		}
		fe.buf.v = nil
		fe.c = cursor{}
		i.state.foreach.Store(file, fe)
		return

	}
	return
}
