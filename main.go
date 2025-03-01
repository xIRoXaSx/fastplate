package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xiroxasx/fastplate/internal/interpreter"
)

func parseFlags() (a interpreter.Options) {
	flag.BoolVar(&a.Indent, "indent", false, "whether to retain indention or not")
	flag.Var(&a.FileBlacklist, "blacklist", "regex to describe which files should not be interpreted")
	flag.Var(&a.FileWhitelist, "whitelist", "regex to describe which files should be interpreted")
	flag.BoolVar(&a.NoStats, "no-stats", false, "do not print stats at the end of the execution")
	flag.BoolVar(&a.Verbose, "verbose", false, "print verbosely")
	flag.StringVar(&a.InPath, "in", "", "the root path")
	flag.StringVar(&a.OutPath, "out", "", "the output path. If not used, in will be overwritten")
	flag.BoolVar(&a.UseCRLF, "crlf", false, "whether to split lines by \\r\\n or \\n")
	flag.Var(&a.VarFilePaths, "var", "the optional var file path.")
	flag.Parse()
	return
}

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	if len(os.Args) == 1 {
		return
	}

	opts := parseFlags()
	if opts.OutPath == "" {
		r := bufio.NewReader(os.Stdin)
		fmt.Printf("Are you sure that you want to overwrite %s? [y/N] ", opts.InPath)
		b, err := r.ReadByte()
		if err != nil {
			log.Fatal().Err(err).Msg("unable to read input")
		}
		if bytes.ToLower([]byte{b})[0] != 'y' {
			log.Fatal().Err(err).Msg("canceled")
		}
		opts.OutPath = opts.InPath
	}

	if opts.InPath == "" {
		log.Fatal().Msg("in path needs to be defined")
	}

	opts.InPath = filepath.Clean(opts.InPath)
	opts.OutPath = filepath.Clean(opts.OutPath)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if opts.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	ip := interpreter.New(&opts)
	err := ip.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("error upon execution")
	}
}
