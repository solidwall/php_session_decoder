package php_serialize

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
)

const UNSERIALIZABLE_OBJECT_MAX_SIZE_DEFAULT = 10 * 1024 * 1024

func UnSerialize(s string) (PhpValue, error) {
	decoder := NewUnSerializer(s)
	decoder.SetSerializedDecodeFunc(SerializedDecodeFunc(UnSerialize))
	return decoder.Decode()
}

type UnSerializer struct {
	source     string
	r          *strings.Reader
	lastErr    error
	decodeFunc SerializedDecodeFunc
	maxSize    int
}

func NewUnSerializer(data string) *UnSerializer {
	return NewUnSerializerWithLimit(data, UNSERIALIZABLE_OBJECT_MAX_SIZE_DEFAULT)
}

func NewUnSerializerWithLimit(data string, limit int) *UnSerializer {
	return &UnSerializer{
		source:  data,
		maxSize: limit,
	}
}

func (us *UnSerializer) SetReader(r *strings.Reader) {
	us.r = r
}

func (us *UnSerializer) SetSerializedDecodeFunc(f SerializedDecodeFunc) {
	us.decodeFunc = f
}

func (us *UnSerializer) Decode() (PhpValue, error) {
	if us.r == nil {
		us.r = strings.NewReader(us.source)
	}

	var value PhpValue

	if token, _, err := us.r.ReadRune(); err == nil {
		switch token {
		default:
			us.saveError(fmt.Errorf("php_serialize: Unknown token %#U", token))
		case TOKEN_NULL:
			value = us.decodeNull()
		case TOKEN_BOOL:
			value = us.decodeBool()
		case TOKEN_INT:
			value = us.decodeNumber(false)
		case TOKEN_FLOAT:
			value = us.decodeNumber(true)
		case TOKEN_STRING:
			value = us.decodeString(DELIMITER_STRING_LEFT, DELIMITER_STRING_RIGHT, true)
		case TOKEN_ARRAY:
			value = us.decodeArray()
		case TOKEN_OBJECT:
			value = us.decodeObject()
		case TOKEN_OBJECT_SERIALIZED:
			value = us.decodeSerialized()
		case TOKEN_REFERENCE, TOKEN_REFERENCE_OBJECT:
			value = us.decodeReference()
		case TOKEN_SPL_ARRAY:
			value = us.decodeSplArray()

		}
	}

	return value, us.lastErr
}

func (us *UnSerializer) decodeNull() PhpValue {
	us.expect(SEPARATOR_VALUES)
	return nil
}

func (us *UnSerializer) decodeBool() PhpValue {
	var (
		raw rune
		err error
	)
	us.expect(SEPARATOR_VALUE_TYPE)

	if raw, _, err = us.r.ReadRune(); err != nil {
		us.saveError(fmt.Errorf("php_serialize: Error while reading bool value: %v", err))
	}

	us.expect(SEPARATOR_VALUES)
	return raw == '1'
}

func (us *UnSerializer) decodeNumber(isFloat bool) PhpValue {
	var (
		raw string
		err error
		val PhpValue
	)
	us.expect(SEPARATOR_VALUE_TYPE)

	if raw, err = us.readUntil(SEPARATOR_VALUES); err != nil {
		us.saveError(fmt.Errorf("php_serialize: Error while reading number value: %v", err))
	} else {
		if isFloat {
			if val, err = strconv.ParseFloat(raw, 64); err != nil {
				us.saveError(fmt.Errorf("php_serialize: Unable to convert %s to float: %v", raw, err))
			}
		} else {
			if val, err = strconv.Atoi(raw); err != nil {
				us.saveError(fmt.Errorf("php_serialize: Unable to convert %s to int: %v", raw, err))
			}
		}
	}

	return val
}

func (us *UnSerializer) decodeString(left, right rune, isFinal bool) PhpValue {
	var (
		err     error
		val     PhpValue
		strLen  int
		readLen int
	)

	strLen = us.readLen()
	us.expect(left)

	if strLen > 0 {
		if strLen > us.maxSize {
			us.saveError(fmt.Errorf("php_serialize: Unserializable object length looks too big(%d). If you are sure you wanna unserialise it, please increase max size limit", val))
		} else {
			buf := make([]byte, strLen)
			if readLen, err = us.r.Read(buf); err != nil {
				us.saveError(fmt.Errorf("php_serialize: Error while reading string value: %v", err))
			} else {
				if readLen != strLen {
					us.saveError(fmt.Errorf("php_serialize: Unable to read string. Expected %d but have got %d bytes", strLen, readLen))
				} else {
					val = string(buf)
				}
			}
		}
	}

	us.expect(right)
	if isFinal {
		us.expect(SEPARATOR_VALUES)
	}
	return val
}

func (us *UnSerializer) decodeArray() PhpValue {
	var arrLen int
	val := make(PhpArray)

	arrLen = us.readLen()
	us.expect(DELIMITER_OBJECT_LEFT)

	for i := 0; i < arrLen; i++ {
		k, errKey := us.Decode()
		v, errVal := us.Decode()

		if errKey == nil && errVal == nil {
			switch t := k.(type) {
			default:
				us.saveError(fmt.Errorf("php_serialize: Unexpected key type %T", t))
			case string, int:
				val[k] = v
			}
		} else {
			us.saveError(fmt.Errorf("php_serialize: Error while reading key or(and) value of array"))
		}
	}

	us.expect(DELIMITER_OBJECT_RIGHT)
	return val
}

func (us *UnSerializer) decodeObject() PhpValue {
	val := &PhpObject{
		className: us.readClassName(),
	}

	rawMembers := us.decodeArray()
	val.members, _ = rawMembers.(PhpArray)

	return val
}

func (us *UnSerializer) decodeSerialized() PhpValue {
	val := &PhpObjectSerialized{
		className: us.readClassName(),
	}

	rawData := us.decodeString(DELIMITER_OBJECT_LEFT, DELIMITER_OBJECT_RIGHT, false)
	val.data, _ = rawData.(string)

	if us.decodeFunc != nil && val.data != "" {
		var err error
		if val.value, err = us.decodeFunc(val.data); err != nil {
			us.saveError(err)
		}
	}

	return val
}

func (us *UnSerializer) decodeReference() PhpValue {
	us.expect(SEPARATOR_VALUE_TYPE)
	if _, err := us.readUntil(SEPARATOR_VALUES); err != nil {
		us.saveError(fmt.Errorf("php_serialize: Error while reading reference value: %v", err))
	}
	return nil
}

func (us *UnSerializer) expect(expected rune) {
	if token, _, err := us.r.ReadRune(); err != nil {
		us.saveError(fmt.Errorf("php_serialize: Error while reading expected rune %#U: %v", expected, err))
	} else if token != expected {
		if debugMode {
			log.Printf("php_serialize: source\n%s\n", us.source)
			log.Printf("php_serialize: reader info\n%#v\n", us.r)
		}
		us.saveError(fmt.Errorf("php_serialize: Expected %#U but have got %#U", expected, token))
	}
}

func (us *UnSerializer) readUntil(stop rune) (string, error) {
	var (
		token rune
		err   error
	)
	buf := bytes.NewBuffer([]byte{})

	for {
		if token, _, err = us.r.ReadRune(); err != nil || token == stop {
			break
		} else {
			buf.WriteRune(token)
		}
	}

	return buf.String(), err
}

func (us *UnSerializer) readLen() int {
	var (
		raw string
		err error
		val int
	)
	us.expect(SEPARATOR_VALUE_TYPE)

	if raw, err = us.readUntil(SEPARATOR_VALUE_TYPE); err != nil {
		us.saveError(fmt.Errorf("php_serialize: Error while reading lenght of value: %v", err))
	} else {
		if val, err = strconv.Atoi(raw); err != nil {
			us.saveError(fmt.Errorf("php_serialize: Unable to convert %s to int: %v", raw, err))
		} else if val > us.maxSize {
			us.saveError(fmt.Errorf("php_serialize: Unserializable object length looks too big(%d). If you are sure you wanna unserialise it, please increase max size limit", val))
			val = 0
		}
	}
	return val
}

func (us *UnSerializer) readClassName() (res string) {
	rawClass := us.decodeString(DELIMITER_STRING_LEFT, DELIMITER_STRING_RIGHT, false)
	res, _ = rawClass.(string)
	return
}

func (us *UnSerializer) saveError(err error) {
	if us.lastErr == nil {
		us.lastErr = err
	}
}

func (us *UnSerializer) decodeSplArray() PhpValue {
	var err error
	val := &PhpSplArray{}

	us.expect(SEPARATOR_VALUE_TYPE)
	us.expect(TOKEN_INT)

	flags := us.decodeNumber(false)
	if flags == nil {
		us.saveError(fmt.Errorf("php_serialize: Unable to read flags of SplArray"))
		return nil
	}
	val.flags = PhpValueInt(flags)

	if val.array, err = us.Decode(); err != nil {
		us.saveError(fmt.Errorf("php_serialize: Can't parse SplArray: %v", err))
		return nil
	}

	us.expect(SEPARATOR_VALUES)
	us.expect(TOKEN_SPL_ARRAY_MEMBERS)
	us.expect(SEPARATOR_VALUE_TYPE)

	if val.properties, err = us.Decode(); err != nil {
		us.saveError(fmt.Errorf("php_serialize: Can't parse properties of SplArray: %v", err))
		return nil
	}

	return val
}
