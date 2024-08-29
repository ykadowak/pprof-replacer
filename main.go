package main

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/ykadowak/pprof-replcacer/pb"
	"google.golang.org/protobuf/proto"
)

const FROM_STRING = "from"
const TO_STRING = "to"

func replaceSymbol(path string, from string, to string) error {
	isGzipped := false
	switch filepath.Ext(path) {
	case ".gz":
		isGzipped = true
	case ".pb":
		break
	default:
		return errors.New("unsupported file format. only .pb or .pb.gz is supported")
	}

	// read src pprof file and decode it into go struct
	inputFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to read the target pprof file: %w", err)
	}
	defer inputFile.Close()

	var reader io.Reader
	reader = inputFile
	if isGzipped {
		reader, err = gzip.NewReader(inputFile)
		if err != nil {
			return err
		}
	}

	inputBytes, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("io.ReadAll: %w", err)
	}

	profile := pb.Profile{}

	if err := proto.Unmarshal(inputBytes, &profile); err != nil {
		return fmt.Errorf("failed to unmarshal the target pprof data: %w", err)
	}

	for i, s := range profile.StringTable {
		if s == from {
			log.Default().Printf("replacing %s to %s\n", from, to)
			profile.StringTable[i] = to
		}
	}

	b, err := proto.Marshal(&profile)
	if err != nil {
		panic(err)
	}

	// TODO: export with gzip?
	base := filepath.Base(path)
	base = strings.Split(base, ".")[0]
	if err := os.WriteFile(fmt.Sprintf("%s_new.pb", base), b, 0777); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     FROM_STRING,
				Aliases:  []string{"f"},
				Required: true,
				Usage:    "Symbol name to be replcaced from",
			},
			&cli.StringFlag{
				Name:     TO_STRING,
				Aliases:  []string{"t"},
				Required: true,
				Usage:    "Symbol name to be replaced to",
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() == 0 {
				return errors.New("needs to specify a target file")
			}

			file := cCtx.Args().Get(0)
			from := cCtx.String(FROM_STRING)
			to := cCtx.String(TO_STRING)
			return replaceSymbol(file, from, to)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
