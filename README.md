# lz4

lz4 implements reading and writing of lz4 format compressed files for Go, following lz4 stream format.
It uses the lz4 C library underneath.

## Installation

Download and install:

```console
$ go get github.com/cyberdelia/lz4
```

Add it to your code:

```go
import "github.com/cyberdelia/lz4"
```

## Command line tool
 
Download and install:

```console
$ go get github.com/cyberdelia/lz4/cmd/lz4
```

Compress and decompress:

```console
$ lz4 testdata/pg135.txt
$ lz4 -d testdata/pg135.txt.lz4
```
