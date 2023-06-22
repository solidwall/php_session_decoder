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
			return nil, fmt.Errorf("php_serialize: Unknown token %#U", token)
		case TOKEN_NULL:
			return us.decodeNull()
		case TOKEN_BOOL:
			return us.decodeBool()
		case TOKEN_INT:
			return us.decodeNumber(false)
		case TOKEN_FLOAT:
			return us.decodeNumber(true)
		case TOKEN_STRING:
			return us.decodeString(DELIMITER_STRING_LEFT, DELIMITER_STRING_RIGHT, true)
		case TOKEN_ARRAY:
			return us.decodeArray()
		case TOKEN_OBJECT:
			return us.decodeObject()
		case TOKEN_OBJECT_SERIALIZED:
			return us.decodeSerialized()
		case TOKEN_REFERENCE, TOKEN_REFERENCE_OBJECT:
			return us.decodeReference()
		case TOKEN_SPL_ARRAY:
			return us.decodeSplArray()
		}
	}

	return value, nil
}

func (us *UnSerializer) decodeNull() (PhpValue, error) {
	return nil, us.expect(SEPARATOR_VALUES)
}

func (us *UnSerializer) decodeBool() (PhpValue, error) {
	var (
		raw rune
		err error
	)
	err = us.expect(SEPARATOR_VALUE_TYPE)
	if err != nil {
		return nil, err
	}

	if raw, _, err = us.r.ReadRune(); err != nil {
		return nil, fmt.Errorf("php_serialize: Error while reading bool value: %v", err)
	}

	err = us.expect(SEPARATOR_VALUES)
	if err != nil {
		return nil, err
	}
	return raw == '1', nil
}

func (us *UnSerializer) decodeNumber(isFloat bool) (PhpValue, error) {
	var (
		raw string
		err error
		val PhpValue
	)
	err = us.expect(SEPARATOR_VALUE_TYPE)
	if err != nil {
		return nil, err
	}

	if raw, err = us.readUntil(SEPARATOR_VALUES); err != nil {
		return nil, fmt.Errorf("php_serialize: Error while reading number value: %v", err)
	} else {
		if isFloat {
			if val, err = strconv.ParseFloat(raw, 64); err != nil {
				return nil, fmt.Errorf("php_serialize: Unable to convert %s to float: %v", raw, err)
			}
		} else {
			if val, err = strconv.Atoi(raw); err != nil {
				return nil, fmt.Errorf("php_serialize: Unable to convert %s to int: %v", raw, err)
			}
		}
	}

	return val, nil
}

func (us *UnSerializer) decodeString(left, right rune, isFinal bool) (PhpValue, error) {
	var (
		err     error
		val     PhpValue
		strLen  int
		readLen int
	)

	strLen, err = us.readLen()
	if err != nil {
		return nil, err
	}
	err = us.expect(left)
	if err != nil {
		return nil, err
	}

	if strLen > 0 {
		if strLen > us.maxSize {
			return nil, fmt.Errorf("php_serialize: Unserializable object length looks too big(%d). If you are sure you wanna unserialise it, please increase max size limit", val)
		} else {
			buf := make([]byte, strLen)
			if readLen, err = us.r.Read(buf); err != nil {
				return nil, fmt.Errorf("php_serialize: Error while reading string value: %v", err)
			} else {
				if readLen != strLen {
					return nil, fmt.Errorf("php_serialize: Unable to read string. Expected %d but have got %d bytes", strLen, readLen)
				} else {
					val = string(buf)
				}
			}
		}
	}

	err = us.expect(right)
	if err != nil {
		return nil, err
	}
	if isFinal {
		err = us.expect(SEPARATOR_VALUES)
		if err != nil {
			return nil, err
		}
	}
	return val, nil
}

func (us *UnSerializer) decodeArray() (PhpValue, error) {
	var (
		arrLen int
		err    error
	)
	val := make(PhpArray)

	arrLen, err = us.readLen()
	if err != nil {
		return nil, err
	}

	err = us.expect(DELIMITER_OBJECT_LEFT)
	if err != nil {
		return nil, err
	}

	for i := 0; i < arrLen; i++ {
		k, errKey := us.Decode()
		v, errVal := us.Decode()

		if errKey == nil && errVal == nil {
			switch t := k.(type) {
			default:
				return nil, fmt.Errorf("php_serialize: Unexpected key type %T", t)
			case string, int:
				val[k] = v
			}
		} else {
			return nil, fmt.Errorf("php_serialize: Error while reading key or(and) value of array")
		}
	}

	err = us.expect(DELIMITER_OBJECT_RIGHT)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (us *UnSerializer) decodeObject() (PhpValue, error) {
	name, err := us.readClassName()
	if err != nil {
		return nil, err
	}

	val := &PhpObject{
		className: name,
	}

	rawMembers, err := us.decodeArray()
	if err != nil {
		return nil, err
	}
	val.members, _ = rawMembers.(PhpArray)

	return val, nil
}

func (us *UnSerializer) decodeSerialized() (PhpValue, error) {
	name, err := us.readClassName()
	if err != nil {
		return nil, err
	}
	val := &PhpObjectSerialized{
		className: name,
	}

	rawData, err := us.decodeString(DELIMITER_OBJECT_LEFT, DELIMITER_OBJECT_RIGHT, false)
	if err != nil {
		return nil, err
	}
	val.data, _ = rawData.(string)

	if us.decodeFunc != nil && val.data != "" {
		var err error
		if val.value, err = us.decodeFunc(val.data); err != nil {
			return nil, err
		}
	}

	return val, nil
}

func (us *UnSerializer) decodeReference() (PhpValue, error) {
	err := us.expect(SEPARATOR_VALUE_TYPE)
	if err != nil {
		return nil, err
	}
	if _, err := us.readUntil(SEPARATOR_VALUES); err != nil {
		return nil, fmt.Errorf("php_serialize: Error while reading reference value: %v", err)
	}
	return nil, nil
}

func (us *UnSerializer) expect(expected rune) error {
	if token, _, err := us.r.ReadRune(); err != nil {
		return fmt.Errorf("php_serialize: Error while reading expected rune %#U: %v", expected, err)
	} else if token != expected {
		if debugMode {
			log.Printf("php_serialize: source\n%s\n", us.source)
			log.Printf("php_serialize: reader info\n%#v\n", us.r)
		}
		return fmt.Errorf("php_serialize: Expected %#U but have got %#U", expected, token)
	}
	return nil
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

func (us *UnSerializer) readLen() (int, error) {
	var (
		raw string
		err error
		val int
	)
	err = us.expect(SEPARATOR_VALUE_TYPE)
	if err != nil {
		return 0, err
	}

	if raw, err = us.readUntil(SEPARATOR_VALUE_TYPE); err != nil {
		return 0, fmt.Errorf("php_serialize: Error while reading lenght of value: %v", err)
	} else {
		if val, err = strconv.Atoi(raw); err != nil {
			return 0, fmt.Errorf("php_serialize: Unable to convert %s to int: %v", raw, err)
		} else if val > us.maxSize {
			return 0, fmt.Errorf("php_serialize: Unserializable object length looks too big(%d). If you are sure you wanna unserialise it, please increase max size limit", val)
		}
	}
	return val, nil
}

func (us *UnSerializer) readClassName() (res string, err error) {
	rawClass, err := us.decodeString(DELIMITER_STRING_LEFT, DELIMITER_STRING_RIGHT, false)
	if err != nil {
		return "", err
	}
	res, _ = rawClass.(string)
	return res, nil
}

func (us *UnSerializer) decodeSplArray() (PhpValue, error) {
	var err error
	val := &PhpSplArray{}

	err = us.expect(SEPARATOR_VALUE_TYPE)
	if err != nil {
		return nil, err
	}
	err = us.expect(TOKEN_INT)
	if err != nil {
		return nil, err
	}

	flags, err := us.decodeNumber(false)
	if err != nil {
		return nil, err
	}
	if flags == nil {
		return nil, fmt.Errorf("php_serialize: Unable to read flags of SplArray")
	}
	val.flags = PhpValueInt(flags)

	if val.array, err = us.Decode(); err != nil {
		return nil, fmt.Errorf("php_serialize: Can't parse SplArray: %v", err)
	}

	err = us.expect(SEPARATOR_VALUES)
	if err != nil {
		return nil, err
	}
	err = us.expect(TOKEN_SPL_ARRAY_MEMBERS)
	if err != nil {
		return nil, err
	}
	err = us.expect(SEPARATOR_VALUE_TYPE)
	if err != nil {
		return nil, err
	}

	if val.properties, err = us.Decode(); err != nil {
		return nil, fmt.Errorf("php_serialize: Can't parse properties of SplArray: %v", err)
	}

	return val, nil
}
