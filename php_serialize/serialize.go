package php_serialize

import (
	"bytes"
	"fmt"
	"strconv"
)

func Serialize(v PhpValue) (string, error) {
	encoder := NewSerializer()
	encoder.SetSerializedEncodeFunc(SerializedEncodeFunc(Serialize))
	return encoder.Encode(v)
}

type Serializer struct {
	lastErr    error
	encodeFunc SerializedEncodeFunc
}

func NewSerializer() *Serializer {
	return &Serializer{}
}

func (s *Serializer) SetSerializedEncodeFunc(f SerializedEncodeFunc) {
	s.encodeFunc = f
}

func (s *Serializer) Encode(v PhpValue) (string, error) {
	var value bytes.Buffer

	switch t := v.(type) {
	default:
		s.saveError(fmt.Errorf("php_serialize: Unknown type %T with value %#v", t, v))
	case nil:
		value = s.encodeNull()
	case bool:
		value = s.encodeBool(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		value = s.encodeNumber(v)
	case string:
		value = s.encodeString(v, DELIMITER_STRING_LEFT, DELIMITER_STRING_RIGHT, true)
	case PhpArray, map[PhpValue]PhpValue, PhpSlice:
		value = s.encodeArray(v, true)
	case *PhpObject:
		value = s.encodeObject(v)
	case *PhpObjectSerialized:
		value = s.encodeSerialized(v)
	case *PhpSplArray:
		value = s.encodeSplArray(v)
	}

	return value.String(), s.lastErr
}

func (s *Serializer) encodeNull() (buffer bytes.Buffer) {
	buffer.WriteRune(TOKEN_NULL)
	buffer.WriteRune(SEPARATOR_VALUES)
	return
}

func (s *Serializer) encodeBool(v PhpValue) (buffer bytes.Buffer) {
	buffer.WriteRune(TOKEN_BOOL)
	buffer.WriteRune(SEPARATOR_VALUE_TYPE)

	if bVal, ok := v.(bool); ok && bVal {
		buffer.WriteString("1")
	} else {
		buffer.WriteString("0")
	}

	buffer.WriteRune(SEPARATOR_VALUES)
	return
}

func (s *Serializer) encodeNumber(v PhpValue) (buffer bytes.Buffer) {
	var val string

	isFloat := false

	switch v := v.(type) {
	default:
		val = "0"
	case int:
		val = strconv.FormatInt(int64(v), 10)
	case int8:
		val = strconv.FormatInt(int64(v), 10)
	case int16:
		val = strconv.FormatInt(int64(v), 10)
	case int32:
		val = strconv.FormatInt(int64(v), 10)
	case int64:
		val = strconv.FormatInt(int64(v), 10)
	case uint:
		val = strconv.FormatUint(uint64(v), 10)
	case uint8:
		val = strconv.FormatUint(uint64(v), 10)
	case uint16:
		val = strconv.FormatUint(uint64(v), 10)
	case uint32:
		val = strconv.FormatUint(uint64(v), 10)
	case uint64:
		val = strconv.FormatUint(uint64(v), 10)
	// PHP has precision = 17 by default
	case float32:
		val = strconv.FormatFloat(float64(v), FORMATTER_FLOAT, FORMATTER_PRECISION, 32)
		isFloat = true
	case float64:
		val = strconv.FormatFloat(float64(v), FORMATTER_FLOAT, FORMATTER_PRECISION, 64)
		isFloat = true
	}

	if isFloat {
		buffer.WriteRune(TOKEN_FLOAT)
	} else {
		buffer.WriteRune(TOKEN_INT)
	}

	buffer.WriteRune(SEPARATOR_VALUE_TYPE)
	buffer.WriteString(val)
	buffer.WriteRune(SEPARATOR_VALUES)

	return
}

func (s *Serializer) encodeString(v PhpValue, left, right rune, isFinal bool) (buffer bytes.Buffer) {
	val, _ := v.(string)

	if isFinal {
		buffer.WriteRune(TOKEN_STRING)
	}

	buffer.WriteString(s.prepareLen(len(val)))
	buffer.WriteRune(left)
	buffer.WriteString(val)
	buffer.WriteRune(right)

	if isFinal {
		buffer.WriteRune(SEPARATOR_VALUES)
	}

	return
}

func (s *Serializer) encodeArray(array PhpValue, isFinal bool) (buffer bytes.Buffer) {
	var (
		arrLen int
		str    string
	)

	if isFinal {
		buffer.WriteRune(TOKEN_ARRAY)
	}

	switch array := array.(type) {
	case PhpArray:
		arrLen = len(array)

		buffer.WriteString(s.prepareLen(arrLen))
		buffer.WriteRune(DELIMITER_OBJECT_LEFT)

		for k, v := range array {
			str, _ = s.Encode(k)
			buffer.WriteString(str)
			str, _ = s.Encode(v)
			buffer.WriteString(str)
		}

	case map[PhpValue]PhpValue:
		arrLen = len(array)

		buffer.WriteString(s.prepareLen(arrLen))
		buffer.WriteRune(DELIMITER_OBJECT_LEFT)

		for k, v := range array {
			str, _ = s.Encode(k)
			buffer.WriteString(str)
			str, _ = s.Encode(v)
			buffer.WriteString(str)
		}
	case PhpSlice:
		arrLen = len(array)

		buffer.WriteString(s.prepareLen(arrLen))
		buffer.WriteRune(DELIMITER_OBJECT_LEFT)

		for k, v := range array {
			str, _ = s.Encode(k)
			buffer.WriteString(str)
			str, _ = s.Encode(v)
			buffer.WriteString(str)
		}
	}

	buffer.WriteRune(DELIMITER_OBJECT_RIGHT)

	return
}

func (s *Serializer) encodeObject(v PhpValue) (buffer bytes.Buffer) {
	obj, _ := v.(*PhpObject)
	buffer.WriteRune(TOKEN_OBJECT)
	buffer.WriteString(s.prepareClassName(obj.className))
	encoded := s.encodeArray(obj.members, false)
	buffer.WriteString(encoded.String())
	return
}

func (s *Serializer) encodeSerialized(v PhpValue) (buffer bytes.Buffer) {
	var serialized string

	obj, _ := v.(*PhpObjectSerialized)
	buffer.WriteRune(TOKEN_OBJECT_SERIALIZED)
	buffer.WriteString(s.prepareClassName(obj.className))

	if s.encodeFunc == nil {
		serialized = obj.GetData()
	} else {
		var err error
		if serialized, err = s.encodeFunc(obj.GetValue()); err != nil {
			s.saveError(err)
		}
	}

	encoded := s.encodeString(serialized, DELIMITER_OBJECT_LEFT, DELIMITER_OBJECT_RIGHT, false)
	buffer.WriteString(encoded.String())
	return
}

func (s *Serializer) encodeSplArray(v PhpValue) bytes.Buffer {
	var buffer bytes.Buffer
	obj, _ := v.(*PhpSplArray)

	buffer.WriteRune(TOKEN_SPL_ARRAY)
	buffer.WriteRune(SEPARATOR_VALUE_TYPE)

	encoded := s.encodeNumber(obj.flags)
	buffer.WriteString(encoded.String())

	data, _ := s.Encode(obj.array)
	buffer.WriteString(data)

	buffer.WriteRune(SEPARATOR_VALUES)
	buffer.WriteRune(TOKEN_SPL_ARRAY_MEMBERS)
	buffer.WriteRune(SEPARATOR_VALUE_TYPE)

	data, _ = s.Encode(obj.properties)
	buffer.WriteString(data)

	return buffer
}

func (s *Serializer) prepareLen(l int) string {
	return string(SEPARATOR_VALUE_TYPE) + strconv.Itoa(l) + string(SEPARATOR_VALUE_TYPE)
}

func (s *Serializer) prepareClassName(name string) string {
	encoded := s.encodeString(name, DELIMITER_STRING_LEFT, DELIMITER_STRING_RIGHT, false)
	return encoded.String()
}

func (s *Serializer) saveError(err error) {
	if s.lastErr == nil {
		s.lastErr = err
	}
}
