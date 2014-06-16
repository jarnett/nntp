package nntp

import (
	"bufio"
	"compress/zlib"
	"io"
	"fmt"
	"strings"
)

type zlibDotResponse struct {
	Reader     *bufio.Reader
	clear      *bufio.Reader
	z          io.ReadCloser
	dotWasRead bool
}

func newZlibDotResponse(clear *bufio.Reader) (*zlibDotResponse, error) {
	z, zerr := zlib.NewReader(clear)
	if zerr != nil {
		return nil, zerr
	}

	return &zlibDotResponse{clear: clear, z: z, Reader: bufio.NewReader(z)}, nil
}

func (z *zlibDotResponse) Close() error {
	z.z.Close()

	return z.readDot()
}

func (z *zlibDotResponse) readDot() error {
	if z.dotWasRead {
		return nil
	}

	z.dotWasRead = true

	if line, err := z.clear.ReadString('\n'); err != nil {
		return err
	} else if strings.TrimRight(line, "\r\n") != "." {
		return ProtocolError(fmt.Sprintf(`expected "." on a line, got %+q`, line))
	} else {
		return nil
	}
}
