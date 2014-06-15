package nntp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type yencReader struct {
	r               *bufio.Reader
	readHeader      bool
	awaitingSpecial bool
	eof             bool
	buf             []byte
}

func (y *yencReader) Read(p []byte) (n int, err error) {
	if len(y.buf) == 0 {
		if err = y.nextLine(); err != nil {
			return
		}
	}

	n = copy(p, y.buf)
	y.buf = y.buf[n:]
	return
}

func (y *yencReader) Close() {
	// read until EOF
	for {
		if y.eof {
			break
		}
		
		y.nextLine()
	}
}

func (y *yencReader) nextLine() error {
	if y.eof {
		return io.EOF
	}

	// read a whole line
	line, err := y.r.ReadBytes('\n')
	if err != nil {
		return err
	}

	// chomp the line ending
	line = bytes.TrimRight(line, "\r\n")

	if !y.readHeader {
		// expect a =ybegin line
		if len(line) >= 7 && string(line[:7]) == "=ybegin" {
			y.readHeader = true
			return nil
		} else {
			return fmt.Errorf("expected =ybegin, got %q", string(line))
		}
	}

	// are we at the end of the yenc blob?
	if len(line) >= 5 && string(line[:5]) == "=yend" {
		// remember this and signal the caller
		y.eof = true
		return io.EOF
	}

	y.buf = y.decode(line)
	return nil
}

func (y *yencReader) decode(line []byte) []byte {
	i, j := 0, 0
	for ; i < len(line); i, j = i+1, j+1 {
		// escaped chars yenc42+yenc64
		if y.awaitingSpecial {
			line[j] = (((line[i] - 42) & 255) - 64) & 255
			y.awaitingSpecial = false
			// if escape char - then skip and backtrack j
		} else if line[i] == '=' {
			y.awaitingSpecial = true
			j--
			continue
			// normal char, yenc42
		} else {
			line[j] = (line[i] - 42) & 255
		}
	}
	// return the new (possibly shorter) slice
	// shorter because of the escaped chars
	return line[:len(line)-(i-j)]
}
