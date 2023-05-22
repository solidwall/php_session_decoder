package php_serialize

import (
	"strconv"
)

func PhpValueString(p PhpValue) (res string) {
	res, _ = p.(string)
	return
}

func PhpValueBool(p PhpValue) (res bool) {
	switch p := p.(type) {
	case bool:
		res = p
	case string:
		res, _ = strconv.ParseBool(p)
	}
	return
}

func PhpValueInt(p PhpValue) (res int) {
	switch p := p.(type) {
	case int:
		res = p
	case int8:
		res = int(p)
	case int16:
		res = int(p)
	case int32:
		res = int(p)
	case int64:
		res = int(p)
	case uint:
		res = int(p)
	case uint8:
		res = int(p)
	case uint16:
		res = int(p)
	case uint32:
		res = int(p)
	case uint64:
		res = int(p)
	case string:
		res, _ = strconv.Atoi(p)
	}
	return
}

func PhpValueInt64(p PhpValue) (res int64) {
	switch p := p.(type) {
	case int64:
		res = p
	default:
		res = int64(PhpValueInt(p))
	}
	return
}

func PhpValueUInt(p PhpValue) (res uint) {
	switch p := p.(type) {
	case uint:
		res = p
	default:
		res = uint(PhpValueInt(p))
	}
	return
}

func PhpValueUInt64(p PhpValue) (res uint64) {
	switch p := p.(type) {
	case uint64:
		res = p
	default:
		res = uint64(PhpValueInt(p))
	}
	return
}

func PhpValueFloat64(p PhpValue) (res float64) {
	switch p := p.(type) {
	case float64:
		res = p
	case string:
		res, _ = strconv.ParseFloat(p, 64)
	default:
		return float64(PhpValueInt(p))
	}
	return
}
