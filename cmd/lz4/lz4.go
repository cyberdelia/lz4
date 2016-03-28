package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"

	"github.com/cyberdelia/lz4"
)

var (
	uncompress = flag.Bool("d", false, "Decompress.")
	level      = flag.Int("l", 3, "Compression level.")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile = flag.String("memprofile", "", "write memory profile to this file")
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
	} else {
		level = lz4.BestSpeed
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

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	path := flag.Arg(0)
	if path == "" {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	if *uncompress == true {
		err = decompress(path)
	} else {
		err = compress(*level, path)
	}
	if err != nil {
		log.Println("lz4:", err)
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
	}
}
