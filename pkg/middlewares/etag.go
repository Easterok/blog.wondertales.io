package middlewares

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	EtagConfig struct {
		Skipper middleware.Skipper
		Weak    bool
		HashFn  func(config EtagConfig) hash.Hash
	}
)

var (
	DefaultEtagConfig = EtagConfig{
		Skipper: middleware.DefaultSkipper,
		Weak:    true,
		HashFn: func(config EtagConfig) hash.Hash {
			if config.Weak {
				const crcPol = 0xD5828281
				crc32qTable := crc32.MakeTable(crcPol)
				return crc32.New(crc32qTable)
			}
			return sha1.New()
		},
	}
	normalizedETagName        = http.CanonicalHeaderKey("Etag")
	normalizedIfNoneMatchName = http.CanonicalHeaderKey("If-None-Match")
	weakPrefix                = "W/"
)

func Etag() echo.MiddlewareFunc {
	return EtagWithConfig(DefaultEtagConfig)
}

func EtagWithConfig(config EtagConfig) echo.MiddlewareFunc {

	if config.Skipper == nil {
		config.Skipper = DefaultEtagConfig.Skipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			skipper := config.Skipper

			if skipper(c) {
				return next(c)
			}

			hashFn := config.HashFn
			if hashFn == nil {
				hashFn = DefaultEtagConfig.HashFn
			}

			originalWriter := c.Response().Writer
			res := c.Response()
			req := c.Request()
			hw := bufferedWriter{rw: res.Writer, hash: hashFn(config), buf: bytes.NewBuffer(nil)}
			res.Writer = &hw
			err = next(c)
			res.Writer = originalWriter
			if err != nil {
				return err
			}

			resHeader := res.Header()

			if hw.hash == nil ||
				resHeader.Get(normalizedETagName) != "" ||
				strconv.Itoa(hw.status)[0] != '2' ||
				hw.status == http.StatusNoContent ||
				hw.buf.Len() == 0 {
				writeRaw(originalWriter, hw)
				return
			}

			etag := fmt.Sprintf("%v-%v", strconv.Itoa(hw.len),
				hex.EncodeToString(hw.hash.Sum(nil)))

			if config.Weak {
				etag = weakPrefix + etag
			}

			resHeader.Set(normalizedETagName, etag)

			ifNoneMatch := req.Header.Get(normalizedIfNoneMatchName) // get the If-None-Match header
			headerFresh := ifNoneMatch == etag || ifNoneMatch == weakPrefix+etag

			if headerFresh {
				originalWriter.WriteHeader(http.StatusNotModified)
				originalWriter.Write(nil)
			} else {
				writeRaw(originalWriter, hw)
			}
			return
		}
	}
}

type bufferedWriter struct {
	rw     http.ResponseWriter
	hash   hash.Hash
	buf    *bytes.Buffer
	len    int
	status int
}

func (hw bufferedWriter) Header() http.Header {
	return hw.rw.Header()
}

func (hw *bufferedWriter) WriteHeader(status int) {
	hw.status = status
}

func (hw *bufferedWriter) Write(b []byte) (int, error) {
	if hw.status == 0 {
		hw.status = http.StatusOK
	}
	l, err := hw.buf.Write(b)
	if err != nil {
		return l, err
	}
	l, err = hw.hash.Write(b)
	hw.len += l
	return l, err
}

func writeRaw(res http.ResponseWriter, hw bufferedWriter) {
	res.WriteHeader(hw.status)
	res.Write(hw.buf.Bytes())
}
