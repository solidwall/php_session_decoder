package main

import (
	"bytes"
	"fmt"

	"github.com/solidwall/php_session_decoder/php_serialize"
)

type PhpEncoder struct {
	data    PhpSession
	encoder *php_serialize.Serializer
}

func NewPhpEncoder(data PhpSession) *PhpEncoder {
	return &PhpEncoder{
		data:    data,
		encoder: php_serialize.NewSerializer(),
	}
}

func (pe *PhpEncoder) SetSerializedEncodeFunc(f php_serialize.SerializedEncodeFunc) {
	pe.encoder.SetSerializedEncodeFunc(f)
}

func (pe *PhpEncoder) Encode() (string, error) {
	if pe.data == nil {
		return "", nil
	}
	var (
		err error
		val string
	)
	buf := bytes.NewBuffer([]byte{})

	for k, v := range pe.data {
		buf.WriteString(k)
		buf.WriteRune(SEPARATOR_VALUE_NAME)
		if val, err = pe.encoder.Encode(v); err != nil {
			err = fmt.Errorf("php_session: error during encode value for %q: %v", k, err)
			break
		}
		buf.WriteString(val)
	}

	return buf.String(), err
}
