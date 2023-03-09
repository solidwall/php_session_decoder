package main

import (
	"bytes"
	"io"
	"strings"

	"github.com/solidwall/php_session_decoder/php_serialize"
)

type PhpDecoder struct {
	source  *strings.Reader
	decoder *php_serialize.UnSerializer
}

func NewPhpDecoder(phpSession string) *PhpDecoder {
	decoder := &PhpDecoder{
		source:  strings.NewReader(phpSession),
		decoder: php_serialize.NewUnSerializer(""),
	}
	decoder.decoder.SetReader(decoder.source)
	return decoder
}

func (pd *PhpDecoder) SetSerializedDecodeFunc(f php_serialize.SerializedDecodeFunc) {
	pd.decoder.SetSerializedDecodeFunc(f)
}

func (pd *PhpDecoder) Decode() (PhpSession, error) {
	var (
		name  string
		err   error
		value php_serialize.PhpValue
	)
	res := make(PhpSession)

	for {
		if name, err = pd.readName(); err != nil {
			break
		}
		if value, err = pd.decoder.Decode(); err != nil {
			break
		}
		res[name] = value
	}

	if err == io.EOF {
		err = nil
	}
	return res, err
}

func (pd *PhpDecoder) readName() (string, error) {
	var (
		token rune
		err   error
	)
	buf := bytes.NewBuffer([]byte{})
	for {
		if token, _, err = pd.source.ReadRune(); err != nil || token == SEPARATOR_VALUE_NAME {
			break
		} else {
			buf.WriteRune(token)
		}
	}
	return buf.String(), err
}
