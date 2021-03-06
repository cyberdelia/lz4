package lz4

import (
	"bytes"
	"io"
	"io/ioutil"
	"runtime"
	"testing"
	"testing/quick"
)

type lz4Test struct {
	name string
	raw  string
	lz4  []byte
	err  error
}

var lz4Tests = []lz4Test{
	{
		"empty",
		"",
		[]byte{
			0x4, 0x22, 0x4d, 0x18, 0x64, 0x70, 0xb9,
			0x0, 0x0, 0x0, 0x0, 0x5, 0x5d, 0xcc, 0x2,
		},
		nil,
	},
	{
		"hello",
		"hello world\n",
		[]byte{
			0x4, 0x22, 0x4d, 0x18, 0x64, 0x70, 0xb9, 0xc, 0x0, 0x0, 0x80,
			0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c,
			0x64, 0xa, 0x0, 0x0, 0x0, 0x0, 0xb0, 0x8d, 0x52, 0xa4,
		},
		nil,
	},
	{
		"hello x2",
		"hello world\nhello world\n",
		[]byte{
			0x4, 0x22, 0x4d, 0x18, 0x64, 0x70, 0xb9, 0x18, 0x0, 0x0, 0x80,
			0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64,
			0xa, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c,
			0x64, 0xa, 0x0, 0x0, 0x0, 0x0, 0xad, 0x2a, 0xaf, 0xc2,
		},
		nil,
	},
	{
		"shesells",
		"she sells seashells by the seashore\n",
		[]byte{
			0x4, 0x22, 0x4d, 0x18, 0x64, 0x70, 0xb9, 0x24, 0x0, 0x0, 0x80, 0x73,
			0x68, 0x65, 0x20, 0x73, 0x65, 0x6c, 0x6c, 0x73, 0x20, 0x73, 0x65,
			0x61, 0x73, 0x68, 0x65, 0x6c, 0x6c, 0x73, 0x20, 0x62, 0x79, 0x20,
			0x74, 0x68, 0x65, 0x20, 0x73, 0x65, 0x61, 0x73, 0x68, 0x6f, 0x72,
			0x65, 0xa, 0x0, 0x0, 0x0, 0x0, 0x40, 0x28, 0x14, 0x89,
		},
		nil,
	},
	{
		"gettysburg",
		"  Four score and seven years ago our fathers brought forth on\n" +
			"this continent, a new nation, conceived in Liberty, and dedicated\n" +
			"to the proposition that all men are created equal.\n" +
			"  Now we are engaged in a great Civil War, testing whether that\n" +
			"nation, or any nation so conceived and so dedicated, can long\n" +
			"endure.\n" +
			"  We are met on a great battle-field of that war.\n" +
			"  We have come to dedicate a portion of that field, as a final\n" +
			"resting place for those who here gave their lives that that\n" +
			"nation might live.  It is altogether fitting and proper that\n" +
			"we should do this.\n" +
			"  But, in a larger sense, we can not dedicate — we can not\n" +
			"consecrate — we can not hallow — this ground.\n" +
			"  The brave men, living and dead, who struggled here, have\n" +
			"consecrated it, far above our poor power to add or detract.\n" +
			"The world will little note, nor long remember what we say here,\n" +
			"but it can never forget what they did here.\n" +
			"  It is for us the living, rather, to be dedicated here to the\n" +
			"unfinished work which they who fought here have thus far so\n" +
			"nobly advanced.  It is rather for us to be here dedicated to\n" +
			"the great task remaining before us — that from these honored\n" +
			"dead we take increased devotion to that cause for which they\n" +
			"gave the last full measure of devotion —\n" +
			"  that we here highly resolve that these dead shall not have\n" +
			"died in vain — that this nation, under God, shall have a new\n" +
			"birth of freedom — and that government of the people, by the\n" +
			"people, for the people, shall not perish from this earth.\n" +
			"\n" +
			"Abraham Lincoln, November 19, 1863, Gettysburg, Pennsylvania\n",
		[]byte{
			0x4, 0x22, 0x4d, 0x18, 0x64, 0x70, 0xb9, 0xc7, 0x4, 0x0, 0x0, 0xf0, 0x12, 0x20,
			0x20, 0x46, 0x6f, 0x75, 0x72, 0x20, 0x73, 0x63, 0x6f, 0x72, 0x65, 0x20, 0x61,
			0x6e, 0x64, 0x20, 0x73, 0x65, 0x76, 0x65, 0x6e, 0x20, 0x79, 0x65, 0x61, 0x72,
			0x73, 0x20, 0x61, 0x67, 0x6f, 0x20, 0x1e, 0x0, 0xf0, 0x27, 0x66, 0x61, 0x74,
			0x68, 0x65, 0x72, 0x73, 0x20, 0x62, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x74, 0x20,
			0x66, 0x6f, 0x72, 0x74, 0x68, 0x20, 0x6f, 0x6e, 0xa, 0x74, 0x68, 0x69, 0x73,
			0x20, 0x63, 0x6f, 0x6e, 0x74, 0x69, 0x6e, 0x65, 0x6e, 0x74, 0x2c, 0x20, 0x61,
			0x20, 0x6e, 0x65, 0x77, 0x20, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2c, 0x19,
			0x0, 0xf1, 0x3, 0x63, 0x65, 0x69, 0x76, 0x65, 0x64, 0x20, 0x69, 0x6e, 0x20,
			0x4c, 0x69, 0x62, 0x65, 0x72, 0x74, 0x79, 0x2c, 0x65, 0x0, 0xf0, 0x9, 0x64,
			0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x64, 0xa, 0x74, 0x6f, 0x20, 0x74,
			0x68, 0x65, 0x20, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x69, 0x38, 0x0, 0xf0,
			0x6, 0x20, 0x74, 0x68, 0x61, 0x74, 0x20, 0x61, 0x6c, 0x6c, 0x20, 0x6d, 0x65,
			0x6e, 0x20, 0x61, 0x72, 0x65, 0x20, 0x63, 0x72, 0x65, 0x2c, 0x0, 0xf1, 0x1,
			0x20, 0x65, 0x71, 0x75, 0x61, 0x6c, 0x2e, 0xa, 0x20, 0x20, 0x4e, 0x6f, 0x77,
			0x20, 0x77, 0x65, 0x1c, 0x0, 0x52, 0x65, 0x6e, 0x67, 0x61, 0x67, 0x62, 0x0,
			0x30, 0x61, 0x20, 0x67, 0x29, 0x0, 0xf0, 0x8, 0x20, 0x43, 0x69, 0x76, 0x69,
			0x6c, 0x20, 0x57, 0x61, 0x72, 0x2c, 0x20, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e,
			0x67, 0x20, 0x77, 0x68, 0x65, 0xc2, 0x0, 0x1, 0x5b, 0x0, 0x14, 0xa, 0x9f, 0x0,
			0x63, 0x6f, 0x72, 0x20, 0x61, 0x6e, 0x79, 0xae, 0x0, 0x37, 0x20, 0x73, 0x6f,
			0xb0, 0x0, 0x1, 0x9, 0x1, 0x16, 0x6f, 0xa7, 0x0, 0xf0, 0x2, 0x2c, 0x20, 0x63,
			0x61, 0x6e, 0x20, 0x6c, 0x6f, 0x6e, 0x67, 0xa, 0x65, 0x6e, 0x64, 0x75, 0x72,
			0x65, 0x86, 0x0, 0x12, 0x57, 0x82, 0x0, 0x56, 0x6d, 0x65, 0x74, 0x20, 0x6f,
			0x7e, 0x0, 0xf1, 0x0, 0x62, 0x61, 0x74, 0x74, 0x6c, 0x65, 0x2d, 0x66, 0x69,
			0x65, 0x6c, 0x64, 0x20, 0x6f, 0x66, 0x73, 0x0, 0x43, 0x20, 0x77, 0x61, 0x72,
			0x32, 0x0, 0xb6, 0x68, 0x61, 0x76, 0x65, 0x20, 0x63, 0x6f, 0x6d, 0x65, 0x20,
			0x74, 0x60, 0x0, 0x61, 0x20, 0x61, 0x20, 0x70, 0x6f, 0x72, 0xfd, 0x0, 0x4, 0x32,
			0x0, 0x1, 0x40, 0x0, 0xe3, 0x2c, 0x20, 0x61, 0x73, 0x20, 0x61, 0x20, 0x66, 0x69,
			0x6e, 0x61, 0x6c, 0xa, 0x72, 0xcc, 0x0, 0x50, 0x70, 0x6c, 0x61, 0x63, 0x65, 0x83,
			0x1, 0xf0, 0x2, 0x20, 0x74, 0x68, 0x6f, 0x73, 0x65, 0x20, 0x77, 0x68, 0x6f, 0x20,
			0x68, 0x65, 0x72, 0x65, 0x20, 0x67, 0x5b, 0x0, 0xb2, 0x74, 0x68, 0x65, 0x69, 0x72,
			0x20, 0x6c, 0x69, 0x76, 0x65, 0x73, 0x7b, 0x0, 0x0, 0x4e, 0x1, 0x3, 0xf3, 0x0, 0x30,
			0x20, 0x6d, 0x69, 0xbe, 0x1, 0x0, 0x1d, 0x0, 0xe2, 0x2e, 0x20, 0x20, 0x49, 0x74,
			0x20, 0x69, 0x73, 0x20, 0x61, 0x6c, 0x74, 0x6f, 0x67, 0x1d, 0x1, 0x31, 0x66, 0x69,
			0x74, 0x2d, 0x1, 0x0, 0xfd, 0x0, 0x0, 0x90, 0x1, 0x4, 0x30, 0x1, 0xb0, 0x77, 0x65,
			0x20, 0x73, 0x68, 0x6f, 0x75, 0x6c, 0x64, 0x20, 0x64, 0xad, 0x1, 0x20, 0x69, 0x73,
			0xcb, 0x0, 0x40, 0x42, 0x75, 0x74, 0x2c, 0xd7, 0x1, 0xf0, 0x0, 0x61, 0x20, 0x6c,
			0x61, 0x72, 0x67, 0x65, 0x72, 0x20, 0x73, 0x65, 0x6e, 0x73, 0x65, 0x2c, 0x97, 0x1,
			0x0, 0x2b, 0x1, 0x35, 0x6e, 0x6f, 0x74, 0x3e, 0x1, 0x47, 0x20, 0xe2, 0x80, 0x94,
			0x18, 0x0, 0x8e, 0xa, 0x63, 0x6f, 0x6e, 0x73, 0x65, 0x63, 0x72, 0x1a, 0x0, 0x71,
			0x20, 0x68, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x30, 0x0, 0x0, 0x68, 0x0, 0x70, 0x20,
			0x67, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x6f, 0x0, 0x60, 0x54, 0x68, 0x65, 0x20, 0x62,
			0x72, 0xe1, 0x0, 0x40, 0x6d, 0x65, 0x6e, 0x2c, 0xe0, 0x0, 0x0, 0xd7, 0x1, 0x0,
			0xaa, 0x0, 0x51, 0x64, 0x65, 0x61, 0x64, 0x2c, 0x5, 0x1, 0x91, 0x73, 0x74, 0x72,
			0x75, 0x67, 0x67, 0x6c, 0x65, 0x64, 0xf, 0x1, 0x11, 0x2c, 0x6b, 0x1, 0x7, 0x6d, 0x0,
			0xf1, 0x0, 0x64, 0x20, 0x69, 0x74, 0x2c, 0x20, 0x66, 0x61, 0x72, 0x20, 0x61, 0x62,
			0x6f, 0x76, 0x65, 0xd9, 0x2, 0x30, 0x70, 0x6f, 0x6f, 0x5, 0x0, 0x10, 0x77, 0xeb, 0x0,
			0xf0, 0x3, 0x6f, 0x20, 0x61, 0x64, 0x64, 0x20, 0x6f, 0x72, 0x20, 0x64, 0x65, 0x74,
			0x72, 0x61, 0x63, 0x74, 0x2e, 0xa, 0x75, 0x0, 0xd0, 0x77, 0x6f, 0x72, 0x6c, 0x64,
			0x20, 0x77, 0x69, 0x6c, 0x6c, 0x20, 0x6c, 0x69, 0xda, 0x1, 0xa1, 0x20, 0x6e, 0x6f,
			0x74, 0x65, 0x2c, 0x20, 0x6e, 0x6f, 0x72, 0x10, 0x2, 0xb0, 0x20, 0x72, 0x65, 0x6d,
			0x65, 0x6d, 0x62, 0x65, 0x72, 0x20, 0x77, 0xb7, 0x2, 0x0, 0x2c, 0x1, 0x22, 0x61,
			0x79, 0x81, 0x0, 0x71, 0xa, 0x62, 0x75, 0x74, 0x20, 0x69, 0x74, 0x3b, 0x2, 0x50,
			0x6e, 0x65, 0x76, 0x65, 0x72, 0xb5, 0x1, 0x32, 0x67, 0x65, 0x74, 0x2a, 0x0, 0x72,
			0x74, 0x68, 0x65, 0x79, 0x20, 0x64, 0x69, 0xad, 0x0, 0x0, 0xe3, 0x0, 0x2, 0x90, 0x1,
			0x50, 0x66, 0x6f, 0x72, 0x20, 0x75, 0xbb, 0x1, 0x13, 0x65, 0xe5, 0x0, 0x31, 0x2c,
			0x20, 0x72, 0x7e, 0x3, 0x10, 0x2c, 0xa2, 0x0, 0x25, 0x62, 0x65, 0x56, 0x1, 0x2, 0x39,
			0x0, 0x0, 0x15, 0x0, 0xf0, 0xa, 0x74, 0x68, 0x65, 0xa, 0x75, 0x6e, 0x66, 0x69, 0x6e,
			0x69, 0x73, 0x68, 0x65, 0x64, 0x20, 0x77, 0x6f, 0x72, 0x6b, 0x20, 0x77, 0x68, 0x69,
			0x63, 0x68, 0x5a, 0x3, 0x11, 0x79, 0x1b, 0x1, 0x12, 0x66, 0xb8, 0x3, 0x1, 0x27, 0x2,
			0x0, 0x17, 0x1, 0x51, 0x20, 0x74, 0x68, 0x75, 0x73, 0xc, 0x1, 0xf5, 0x2, 0x73, 0x6f,
			0xa, 0x6e, 0x6f, 0x62, 0x6c, 0x79, 0x20, 0x61, 0x64, 0x76, 0x61, 0x6e, 0x63, 0x65,
			0x64, 0x1a, 0x2, 0x2, 0x77, 0x0, 0x0, 0xb4, 0x0, 0x1, 0x91, 0x0, 0x1, 0x7d, 0x0, 0x1,
			0x41, 0x0, 0x5, 0x16, 0x3, 0x40, 0x20, 0x74, 0x6f, 0xa, 0xa9, 0x0, 0x2, 0x77, 0x3,
			0x40, 0x74, 0x61, 0x73, 0x6b, 0xe, 0x1, 0x30, 0x61, 0x69, 0x6e, 0x9c, 0x1, 0x30, 0x62,
			0x65, 0x66, 0x53, 0x4, 0x23, 0x75, 0x73, 0xcf, 0x1, 0x70, 0x61, 0x74, 0x20, 0x66, 0x72,
			0x6f, 0x6d, 0x94, 0x0, 0xb0, 0x73, 0x65, 0x20, 0x68, 0x6f, 0x6e, 0x6f, 0x72, 0x65, 0x64,
			0xa, 0xbe, 0x1, 0x0, 0x1c, 0x2, 0xd0, 0x74, 0x61, 0x6b, 0x65, 0x20, 0x69, 0x6e, 0x63,
			0x72, 0x65, 0x61, 0x73, 0x65, 0xd5, 0x1, 0x21, 0x76, 0x6f, 0xf, 0x3, 0x1, 0x1f, 0x4,
			0x10, 0x61, 0x44, 0x1, 0x22, 0x75, 0x73, 0xf5, 0x2, 0x6, 0xda, 0x0, 0x20, 0xa, 0x67,
			0x10, 0x2, 0x0, 0x81, 0x0, 0x71, 0x6c, 0x61, 0x73, 0x74, 0x20, 0x66, 0x75, 0x33, 0x4,
			0x50, 0x61, 0x73, 0x75, 0x72, 0x65, 0x79, 0x3, 0x5, 0x44, 0x0, 0x52, 0xe2, 0x80, 0x94,
			0xa, 0x20, 0xd, 0x3, 0x13, 0x77, 0xc3, 0x0, 0xc1, 0x68, 0x69, 0x67, 0x68, 0x6c, 0x79,
			0x20, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0xe, 0x1, 0x2, 0x91, 0x1, 0x10, 0x73, 0xdd, 0x0,
			0x40, 0x61, 0x64, 0x20, 0x73, 0x89, 0x2, 0x0, 0xe9, 0x1, 0x2, 0x44, 0x2, 0x22, 0x64,
			0x69, 0xc3, 0x4, 0x31, 0x76, 0x61, 0x69, 0x4e, 0x0, 0x2, 0x4c, 0x0, 0x1, 0xa2, 0x2,
			0x4, 0x4c, 0x4, 0xa3, 0x75, 0x6e, 0x64, 0x65, 0x72, 0x20, 0x47, 0x6f, 0x64, 0x2c, 0x3d,
			0x0, 0x1, 0x66, 0x1, 0x1, 0xf, 0x5, 0x31, 0xa, 0x62, 0x69, 0x2e, 0x5, 0x91, 0x66, 0x20,
			0x66, 0x72, 0x65, 0x65, 0x64, 0x6f, 0x6d, 0x11, 0x1, 0x0, 0xbb, 0x2, 0x1, 0x77, 0x0,
			0xa0, 0x67, 0x6f, 0x76, 0x65, 0x72, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x23, 0x0, 0x0, 0xce,
			0x0, 0xa0, 0x70, 0x65, 0x6f, 0x70, 0x6c, 0x65, 0x2c, 0x20, 0x62, 0x79, 0x2d, 0x1, 0x14,
			0xa, 0xf, 0x0, 0x0, 0x1a, 0x2, 0x8, 0x1f, 0x0, 0x6, 0xa5, 0x0, 0x64, 0x70, 0x65, 0x72,
			0x69, 0x73, 0x68, 0x5f, 0x1, 0xf2, 0x10, 0x69, 0x73, 0x20, 0x65, 0x61, 0x72, 0x74, 0x68,
			0x2e, 0xa, 0xa, 0x41, 0x62, 0x72, 0x61, 0x68, 0x61, 0x6d, 0x20, 0x4c, 0x69, 0x6e, 0x63,
			0x6f, 0x6c, 0x6e, 0x2c, 0x20, 0x4e, 0x6f, 0x76, 0xad, 0x2, 0xf0, 0x14, 0x31, 0x39, 0x2c,
			0x20, 0x31, 0x38, 0x36, 0x33, 0x2c, 0x20, 0x47, 0x65, 0x74, 0x74, 0x79, 0x73, 0x62, 0x75,
			0x72, 0x67, 0x2c, 0x20, 0x50, 0x65, 0x6e, 0x6e, 0x73, 0x79, 0x6c, 0x76, 0x61, 0x6e, 0x69,
			0x61, 0xa, 0x0, 0x0, 0x0, 0x0, 0x2c, 0x63, 0xe9, 0x8d,
		},
		nil,
	},
	{
		"hello block checksum",
		"hello world\n",
		[]byte{
			0x4, 0x22, 0x4d, 0x18, 0x74, 0x70, 0x8e, 0xc, 0x0, 0x0, 0x80, 0x68, 0x65, 0x6c,
			0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0xa, 0xb0, 0x8d, 0x52, 0xa4,
			0x0, 0x0, 0x0, 0x0, 0xb0, 0x8d, 0x52, 0xa4,
		},
		nil,
	},
}

func TestDecompressor(t *testing.T) {
	b := new(bytes.Buffer)
	for _, tt := range lz4Tests {
		in := bytes.NewReader(tt.lz4)
		lz4, err := NewReader(in)
		defer func() {
			if lz4.Close() != tt.err {
				t.Errorf("%s: Close: %v want %v", tt.name, err, tt.err)
			}
		}()
		if err != nil {
			t.Errorf("%s: NewReader: %s", tt.name, err)
			continue
		}
		b.Reset()
		n, err := io.Copy(b, lz4)
		if err != tt.err {
			t.Errorf("%s: io.Copy: %v want %v", tt.name, err, tt.err)
		}
		s := b.String()
		if s != tt.raw {
			t.Errorf("%s: got %d-byte %q want %d-byte %q", tt.name, n, s, len(tt.raw), tt.raw)
		}
	}
}

func roundTrip(payload []byte) bool {
	buf := new(bytes.Buffer)

	w := NewWriter(buf)
	if _, err := w.Write(payload); err != nil {
		return false
	}
	w.Close()

	r, err := NewReader(buf)
	if err != nil {
		return false
	}
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return false
	}
	return bytes.Equal(b, payload)
}

func TestRoundTrip(t *testing.T) {
	if err := quick.Check(roundTrip, nil); err != nil {
		t.Error(err)
	}
}

func BenchmarkDecompressor(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	compressed, err := ioutil.ReadFile("testdata/pg135.txt.lz4")
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(compressed)))
	runtime.GC()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r, _ := NewReader(bytes.NewReader(compressed))
		defer r.Close()
		io.Copy(ioutil.Discard, r)
	}
}

func BenchmarkCompressor(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	text, err := ioutil.ReadFile("testdata/pg135.txt")
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(text)))
	runtime.GC()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		w := NewWriter(ioutil.Discard)
		defer w.Close()
		io.Copy(w, bytes.NewReader(text))
	}
}
