package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyberdelia/lz4"
)

var (
	uncompress = flag.Bool("decompress", false, "Decompress.")
	level      = flag.Int("level", 3, "Compression level.")
)

func decompress(path string) error {
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	decompressor, err := lz4.NewReader(input)
	if err != nil {
		return err
	}
	defer decompressor.Close()
	output, err := os.Create(strings.TrimSuffix(path, filepath.Ext(path)))
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, decompressor)
	if err != nil {
		return err
	}
	return nil
}

func compress(level int, path string) error {
	if level > lz4.BestCompression {
		level = lz4.BestCompression
	} else if level < lz4.BestSpeed {
		level = lz4.BestSpeed
	} else {
		level = lz4.DefaultCompression
	}
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	output, err := os.Create(path + ".lz4")
	if err != nil {
		return err
	}
	compressor, err := lz4.NewWriterLevel(output, level)
	defer compressor.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(compressor, input)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	log.SetFlags(0)
	flag.Parse()
	path := flag.Arg(0)
	var err error
	if *uncompress == true {
		err = decompress(path)
	} else {
		err = compress(*level, path)
	}
	if err != nil {
		log.Println("lz4:", err)
	}
}
