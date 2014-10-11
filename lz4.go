package lz4

/*
#cgo LDFLAGS: -llz4
#include "lz4.h"
#include "lz4hc.h"
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"unsafe"

	"github.com/vova616/xxhash"
)

const (
	BestSpeed          = 3
	BestCompression    = 9
	DefaultCompression = -1
	lz4EOM             = uint32(0)
	lz4Magic           = uint32(0x184D2204)
	lz4BlockSizeID     = 7
	lz4BlockSize       = 1 << (8 + (2 * lz4BlockSizeID))
)

var lz4Header = []byte{
	0x4, 0x22, 0x4d, 0x18, 0x64, 0x70, 0xb9,
}

func blockSize(blockID uint32) uint32 {
	return (1 << (8 + (2 * blockID)))
}

type writer struct {
	level      int
	err        error
	compressor func(src []byte, dst []byte, maxSize uint32) (int, error)

	h hash.Hash32
	w io.Writer
}

// NewWriter creates a new Writer that satisfies writes by compressing data
// written to w.
func NewWriter(w io.Writer) io.WriteCloser {
	z, _ := NewWriterLevel(w, DefaultCompression)
	return z
}

// NewWriterLevel is like NewWriter but specifies the compression level instead
// of assuming DefaultCompression.
func NewWriterLevel(w io.Writer, level int) (io.WriteCloser, error) {
	if level < DefaultCompression || level > BestCompression {
		return nil, fmt.Errorf("lz4: invalid compression level: %d", level)
	}
	return &writer{
		level: level,
		w:     w,
		h:     xxhash.New(0),
	}, nil
}

func (z *writer) writeHeader() error {
	// Write magic number and header
	if _, err := z.w.Write(lz4Header); err != nil {
		return err
	}
	return nil
}

func (z *writer) write(v interface{}) error {
	return binary.Write(z.w, binary.LittleEndian, v)
}

// Write writes a compressed form of p to the underlying io.Writer.
func (z *writer) Write(p []byte) (int, error) {
	if z.err != nil {
		return 0, z.err
	}
	// Write headers
	if z.compressor == nil {
		if z.level == BestCompression {
			z.compressor = lz4CompressBest
		} else {
			z.compressor = lz4CompressSpeed
		}
		z.err = z.writeHeader()
		if z.err != nil {
			return 0, z.err
		}
	}

	compressed := make([]byte, lz4BlockSize)
	n, err := z.compressor(p, compressed, lz4BlockSize)
	if err != nil {
		z.err = err
		return 0, z.err
	}

	if n > 0 {
		z.err = z.write(uint32(n))
		if z.err != nil {
			return 0, z.err
		}
		// Write compressed block
		_, z.err = z.w.Write(compressed[0:n])
		if z.err != nil {
			return 0, z.err
		}
	} else {
		z.err = z.write(uint32(len(p)) | 0x80000000)
		if z.err != nil {
			return 0, z.err
		}
		// Write uncompressed block
		_, z.err = z.w.Write(p)
		if z.err != nil {
			return 0, z.err
		}
	}

	if len(p) > 0 {
		z.h.Write(p)
	}

	return len(p), nil
}

// Close closes the Writer. It does not close the underlying io.Writer.
func (z *writer) Close() error {
	z.err = z.write(uint32(0))
	if z.err != nil {
		return z.err
	}
	return z.write(z.h.Sum32())
}

func lz4CompressSpeed(src []byte, dst []byte, maxSize uint32) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	n := C.LZ4_compress_limitedOutput((*C.char)(unsafe.Pointer(&src[0])), (*C.char)(unsafe.Pointer(&dst[0])), C.int(len(src)), C.int(maxSize))
	if n <= 0 {
		return 0, errors.New("lz4: data corruption")
	}
	return int(n), nil
}

func lz4CompressBest(src []byte, dst []byte, maxSize uint32) (int, error) {
	n := C.LZ4_compressHC_limitedOutput((*C.char)(unsafe.Pointer(&src[0])), (*C.char)(unsafe.Pointer(&dst[0])), C.int(len(src)), C.int(maxSize))
	if n <= 0 {
		return 0, errors.New("lz4: data corruption")
	}
	return int(n), nil
}

type reader struct {
	maxBlockSize        uint32
	contentChecksumFlag bool
	blockChecksumFlag   bool

	buf []byte
	r   io.Reader
	h   hash.Hash32
	err error
}

// NewReader creates a new Reader reading the given reader.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	z := &reader{
		r: r,
		h: xxhash.New(0),
	}
	if err := z.readFrame(); err != nil {
		return nil, err
	}
	return z, nil
}

func (z *reader) readFrame() error {
	// Read and check magic
	var magic uint32
	if err := z.read(&magic); err != nil {
		return err
	}
	if magic != lz4Magic {
		return errors.New("lz4: invalid header")
	}
	br := newBitReader(io.LimitReader(z.r, 3))
	version, err := br.ReadBits(2)
	if err != nil {
		return err
	}
	if version != 1 {
		return errors.New("lz4: wrong version number")
	}
	independenceFlag, err := br.ReadBit()
	if err != nil || !independenceFlag {
		return err
	}
	blockChecksumFlag, err := br.ReadBit()
	if err != nil {
		return err
	}
	z.blockChecksumFlag = blockChecksumFlag
	contentSizeFlag, err := br.ReadBit()
	if err != nil || contentSizeFlag {
		return errors.New("lz4: does not support stream size")
	}
	contentChecksumFlag, err := br.ReadBit()
	if err != nil {
		return err
	}
	z.contentChecksumFlag = contentChecksumFlag
	if reserved, err := br.ReadBit(); err != nil || reserved {
		return errors.New("lz4: wrong value for reserved bits")
	}
	dictionaryFlag, err := br.ReadBit()
	if err != nil {
		return err
	}
	if dictionaryFlag {
		return errors.New("lz4: does not support dictionary")
	}
	if reserved, err := br.ReadBit(); err != nil || reserved {
		return errors.New("lz4: wrong value for reserved bits")
	}
	blockMaxSize, err := br.ReadBits(3)
	if err != nil {
		return err
	}
	if blockMaxSize < 4 {
		return errors.New("lz4: unsupported block size")
	}
	z.maxBlockSize = blockSize(blockMaxSize)
	if reserved, err := br.ReadBits(4); err != nil || reserved != 0 {
		return errors.New("lz4: wrong value for reserved bits")
	}
	sum := br.Sum32() >> 8 & 0xFF
	checksum, err := br.ReadBits(8)
	if err != nil {
		return err
	}
	if checksum != sum {
		return errors.New("lz4: stream descriptor error detected")
	}
	z.h.Reset()
	return nil
}

func (z *reader) nextBlock() {
	// Read block size
	var blockSize uint32
	z.err = z.read(&blockSize)
	if z.err != nil {
		return
	}

	uncompressedFlag := (blockSize >> 31) != 0
	blockSize &= 0x7FFFFFFF

	if blockSize == lz4EOM {
		z.err = io.EOF
		return
	}

	if blockSize > z.maxBlockSize {
		z.err = errors.New("lz4: invalid block size")
		return
	}

	// Read block data
	block := make([]byte, blockSize)
	_, z.err = io.ReadFull(z.r, block)
	if z.err != nil {
		return
	}

	if z.blockChecksumFlag {
		// Check block checksum
		var checksum uint32
		z.err = z.read(&checksum)
		if z.err != nil {
			return
		}
		if checksum != xxhash.Checksum32(block) {
			z.err = errors.New("lz4: invalid block checksum detected")
			return
		}
	}

	// Decompress
	data := make([]byte, z.maxBlockSize)
	if !uncompressedFlag {
		n, err := lz4Decompress(block, data, z.maxBlockSize)
		if err != nil {
			z.err = err
			return
		}
		data = data[0:n]
	} else {
		copy(data, block)
		data = data[0:blockSize]
	}

	if z.contentChecksumFlag {
		z.h.Write(data)
	}

	// Add block to our history
	z.buf = append(z.buf, data...)
}

func (z *reader) read(data interface{}) error {
	return binary.Read(z.r, binary.LittleEndian, data)
}

// Read reads a decompressed form of p from the underlying io.Reader.
func (z *reader) Read(p []byte) (int, error) {
	for {
		if len(z.buf) > 0 {
			n := copy(p, z.buf)
			z.buf = z.buf[n:]
			return n, nil
		}
		if z.err != nil {
			return 0, z.err
		}
		z.nextBlock()
	}
}

// Close closes the Reader. It does not close the underlying io.Reader.
func (z *reader) Close() error {
	if z.contentChecksumFlag {
		// Check content checksum
		var checksum uint32
		z.err = z.read(&checksum)
		if z.err != nil || checksum != z.h.Sum32() {
			z.err = errors.New("lz4: invalid content checksum detected")
			return z.err
		}
	}
	if z.err == io.EOF {
		return nil
	}
	return z.err
}

func lz4Decompress(src []byte, dst []byte, maxSize uint32) (int, error) {
	n := C.LZ4_decompress_safe((*C.char)(unsafe.Pointer(&src[0])),
		(*C.char)(unsafe.Pointer(&dst[0])), C.int(len(src)), C.int(maxSize))
	if n < 0 {
		return 0, errors.New("lz4: data corruption")
	}
	return int(n), nil
}
