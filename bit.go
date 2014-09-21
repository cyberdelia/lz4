package lz4

import (
	"bufio"
	"hash"
	"io"

	"github.com/vova616/xxhash"
)

type bitReader struct {
	n    uint32
	bits uint
	err  error
	h    hash.Hash32

	r io.ByteReader
}

// newBitReader returns a new bitReader reading from r. If r is not
// already an io.ByteReader, it will be converted via a bufio.Reader.
func newBitReader(r io.Reader) bitReader {
	r, ok := r.(io.ByteReader)
	if !ok {
		r = bufio.NewReader(r)
	}
	return bitReader{
		r: r,
		h: xxhash.New(0),
	}
}

func (br *bitReader) ReadBits(bits uint) (uint32, error) {
	for bits > br.bits {
		b, err := br.r.ReadByte()
		br.h.Write([]byte{b})
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		if err != nil {
			br.err = err
			return 0, br.err
		}
		br.n <<= 8
		br.n |= uint32(b)
		br.bits += 8
	}
	n := (br.n >> (br.bits - bits)) & ((1 << bits) - 1)
	br.bits -= bits
	return n, nil
}

func (br *bitReader) ReadBit() (bool, error) {
	n, err := br.ReadBits(1)
	return n != 0, err
}

func (br *bitReader) Sum32() uint32 {
	return br.h.Sum32()
}
