package php_serialize

const (
	TOKEN_NULL              rune = 'N'
	TOKEN_BOOL              rune = 'b'
	TOKEN_INT               rune = 'i'
	TOKEN_FLOAT             rune = 'd'
	TOKEN_STRING            rune = 's'
	TOKEN_ARRAY             rune = 'a'
	TOKEN_OBJECT            rune = 'O'
	TOKEN_OBJECT_SERIALIZED rune = 'C'
	TOKEN_REFERENCE         rune = 'R'
	TOKEN_REFERENCE_OBJECT  rune = 'r'
	TOKEN_SPL_ARRAY         rune = 'x'
	TOKEN_SPL_ARRAY_MEMBERS rune = 'm'

	SEPARATOR_VALUE_TYPE rune = ':'
	SEPARATOR_VALUES     rune = ';'

	DELIMITER_STRING_LEFT  rune = '"'
	DELIMITER_STRING_RIGHT rune = '"'
	DELIMITER_OBJECT_LEFT  rune = '{'
	DELIMITER_OBJECT_RIGHT rune = '}'

	FORMATTER_FLOAT     byte = 'g'
	FORMATTER_PRECISION int  = 17
)

var (
	debugMode = false
)

func Debug(value bool) {
	debugMode = value
}

func NewPhpObject(className string) *PhpObject {
	return &PhpObject{
		className: className,
		members:   PhpArray{},
	}
}

type SerializedDecodeFunc func(string) (PhpValue, error)

type SerializedEncodeFunc func(PhpValue) (string, error)

type PhpValue interface{}

type PhpArray map[PhpValue]PhpValue

type PhpSlice []PhpValue

type PhpObject struct {
	className string
	members   PhpArray
}

func (po *PhpObject) GetClassName() string {
	return po.className
}

func (po *PhpObject) SetClassName(name string) *PhpObject {
	po.className = name
	return po
}

func (po *PhpObject) GetMembers() PhpArray {
	return po.members
}

func (po *PhpObject) SetMembers(members PhpArray) *PhpObject {
	po.members = members
	return po
}

func (po *PhpObject) GetPrivate(name string) (v PhpValue, ok bool) {
	v, ok = po.members["\x00"+po.className+"\x00"+name]
	return
}

func (po *PhpObject) SetPrivate(name string, value PhpValue) *PhpObject {
	po.members["\x00"+po.className+"\x00"+name] = value
	return po
}

func (po *PhpObject) GetProtected(name string) (v PhpValue, ok bool) {
	v, ok = po.members["\x00*\x00"+name]
	return
}

func (po *PhpObject) SetProtected(name string, value PhpValue) *PhpObject {
	po.members["\x00*\x00"+name] = value
	return po
}

func (po *PhpObject) GetPublic(name string) (v PhpValue, ok bool) {
	v, ok = po.members[name]
	return
}

func (po *PhpObject) SetPublic(name string, value PhpValue) *PhpObject {
	po.members[name] = value
	return po
}

func NewPhpObjectSerialized(className string) *PhpObjectSerialized {
	return &PhpObjectSerialized{
		className: className,
	}
}

type PhpObjectSerialized struct {
	className string
	data      string
	value     PhpValue
}

func (pos *PhpObjectSerialized) GetClassName() string {
	return pos.className
}

func (pos *PhpObjectSerialized) SetClassName(name string) *PhpObjectSerialized {
	pos.className = name
	return pos
}

func (pos *PhpObjectSerialized) GetData() string {
	return pos.data
}

func (pos *PhpObjectSerialized) SetData(data string) *PhpObjectSerialized {
	pos.data = data
	return pos
}

func (pos *PhpObjectSerialized) GetValue() PhpValue {
	return pos.value
}

func (pos *PhpObjectSerialized) SetValue(value PhpValue) *PhpObjectSerialized {
	pos.value = value
	return pos
}

func NewPhpSplArray(array, properties PhpValue) *PhpSplArray {
	if array == nil {
		array = make(PhpArray)
	}

	if properties == nil {
		properties = make(PhpArray)
	}

	return &PhpSplArray{
		array:      array,
		properties: properties,
	}
}

type PhpSplArray struct {
	flags      int
	array      PhpValue
	properties PhpValue
}

func (psa *PhpSplArray) GetFlags() int {
	return psa.flags
}

func (psa *PhpSplArray) SetFlags(value int) {
	psa.flags = value
}

func (psa *PhpSplArray) GetArray() PhpValue {
	return psa.array
}

func (psa *PhpSplArray) SetArray(value PhpValue) {
	psa.array = value
}

func (psa *PhpSplArray) GetProperties() PhpValue {
	return psa.properties
}

func (psa *PhpSplArray) SetProperties(value PhpValue) {
	psa.properties = value
}
