package phpgo

import (
	"fmt"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf8"
)

const EOF rune = -1

// ============================================================================
// 解析数据

func toRunes(data []byte) []rune {
	n := utf8.RuneCount(data)
	rs := make([]rune, n)
	p := 0
	for i := 0; i < n; i++ {
		r, s := utf8.DecodeRune(data[p:])
		p += s
		rs[i] = r
	}
	return rs
}

type unSerializer struct {
	index int
	runes []rune
}

func newUnSerializer(runes []rune) *unSerializer {
	serializer := &unSerializer{
		index: 0,
		runes: runes,
	}
	return serializer
}

func (serializer *unSerializer) peek(n int) rune {
	i := serializer.index + n
	if i >= len(serializer.runes) {
		return EOF
	}
	return serializer.runes[i]
}

func (serializer *unSerializer) get() rune {
	if serializer.index >= len(serializer.runes) {
		return EOF
	}
	r := serializer.runes[serializer.index]
	serializer.index++
	return r
}

func (serializer *unSerializer) getRange(n int) ([]rune, error) {
	end := serializer.index + n
	if end >= len(serializer.runes) {
		return nil, fmt.Errorf("get runes out of range [%d]", end)
	}
	rs := serializer.runes[serializer.index:end]
	serializer.index = end
	return rs, nil
}

func (serializer *unSerializer) pickNumber(end rune) (int, error) {
	i := 0
	for serializer.peek(i) != end {
		if i == 0 && serializer.peek(i) == '-' {
			i++
			continue
		}
		if !unicode.IsDigit(serializer.peek(i)) {
			return 0, fmt.Errorf("[%d]not a digit", serializer.index)
		}
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("[%d]digit empty", serializer.index)
	}

	s, err := serializer.getRange(i)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(s))
}

func (serializer *unSerializer) pickString() (string, error) {
	if serializer.get() != '"' {
		return "", fmt.Errorf("[%d]not a string start \"", serializer.index)
	}

	i := 0
	for serializer.peek(i) != '"' {
		i++
	}

	s, err := serializer.getRange(i)
	if err != nil {
		return "", err
	}

	if serializer.get() != '"' {
		return "", fmt.Errorf("[%d]not a string end \"", serializer.index)
	}

	return string(s), nil
}

func matchType(serializer *unSerializer) (any, error) {
	t := serializer.peek(0)
	switch t {
	case 'O':
		return matchObject(serializer)
	case 'a':
		return matchArray(serializer)
	case 's':
		return matchString(serializer)
	case 'i':
		return matchInteger(serializer)
	case 'b':
		return matchBool(serializer)
	case 'N':
		return matchNull(serializer)
	default:
		return nil, fmt.Errorf("[%d]unsupported type: %s(%v)", serializer.index, string(t), t)
	}
}

func matchObject(serializer *unSerializer) (any, error) {
	if serializer.get() != 'O' {
		return nil, fmt.Errorf("[%d] not a object type 'O'", serializer.index)
	}
	if serializer.get() != ':' {
		return nil, fmt.Errorf("[%d]not a object tag : ", serializer.index)
	}

	n, err := serializer.pickNumber(':')
	if err != nil {
		return nil, err
	}

	if serializer.get() != ':' {
		return nil, fmt.Errorf("[%d]not a object tag : ", serializer.index)
	}

	s, err := serializer.pickString()
	if err != nil {
		return nil, err
	}

	if len(s) != n {
		return nil, fmt.Errorf("[%d]not a object name len %d != %d", serializer.index, len(s), n)
	}

	if serializer.get() != ':' {
		return nil, fmt.Errorf("[%d]not a object tag : ", serializer.index)
	}

	fieldCount, err := serializer.pickNumber(':')
	if err != nil {
		return nil, err
	}

	if serializer.get() != ':' {
		return nil, fmt.Errorf("[%d]not a object tag : ", serializer.index)
	}

	if serializer.get() != '{' {
		return nil, fmt.Errorf("[%d]not a object tag {", serializer.index)
	}

	result := map[string]any{}
	// 遍历字段
	for range fieldCount {
		// key
		key, err := matchString(serializer)
		if err != nil {
			return nil, err
		}
		if serializer.get() != ';' {
			return nil, fmt.Errorf("[%d]not a object filed key end ;", serializer.index)
		}

		// value
		value, err := matchType(serializer)
		if err != nil {
			return nil, err
		}

		result[key] = value

		if serializer.peek(-1) == '}' {
			continue
		}

		if serializer.get() != ';' {
			return nil, fmt.Errorf("[%d]not a object filed value end ; ", serializer.index)
		}
	}

	if serializer.get() != '}' {
		return nil, fmt.Errorf("[%d]not a object tag }", serializer.index)
	}

	return result, nil
}

func matchArray(serializer *unSerializer) (any, error) {
	if serializer.get() != 'a' {
		return nil, fmt.Errorf("[%d] not a array type 'a'", serializer.index)
	}
	if serializer.get() != ':' {
		return nil, fmt.Errorf("[%d]not a array tag : ", serializer.index)
	}

	n, err := serializer.pickNumber(':')
	if err != nil {
		return nil, err
	}

	if serializer.get() != ':' {
		return nil, fmt.Errorf("[%d]not a array tag : ", serializer.index)
	}

	if serializer.get() != '{' {
		return nil, fmt.Errorf("[%d]not a array tag {", serializer.index)
	}

	c := serializer.peek(0)
	switch c {
	case 'i':
		return matchList(serializer, n)
	case 's':
		return matchMap(serializer, n)
	default:
		return nil, fmt.Errorf("[%d]not a array key start type '%s' ", serializer.index, string(c))
	}
}

func matchList(serializer *unSerializer, n int) ([]any, error) {
	result := make([]any, n)
	for range n {
		// index
		index, err := matchInteger(serializer)
		if err != nil {
			return nil, err
		}
		if serializer.get() != ';' {
			return nil, fmt.Errorf("[%d]not a list index end ;", serializer.index)
		}

		// value
		value, err := matchType(serializer)
		if err != nil {
			return nil, err
		}

		if serializer.get() != ';' {
			return nil, fmt.Errorf("[%d]not a list value end ;", serializer.index)
		}

		result[index] = value
	}

	if serializer.get() != '}' {
		return nil, fmt.Errorf("[%d]not a list tag }", serializer.index)
	}

	return result, nil
}

func matchMap(serializer *unSerializer, n int) (map[string]any, error) {
	result := map[string]any{}
	for range n {
		// key
		key, err := matchString(serializer)
		if err != nil {
			return nil, err
		}
		if serializer.get() != ';' {
			return nil, fmt.Errorf("[%d]not a map filed key end ;", serializer.index)
		}

		// value
		value, err := matchType(serializer)
		if err != nil {
			return nil, err
		}

		if serializer.get() != ';' {
			return nil, fmt.Errorf("[%d]not a map filed value end ;", serializer.index)
		}

		result[key] = value
	}

	if serializer.get() != '}' {
		return nil, fmt.Errorf("[%d]not a map tag }", serializer.index)
	}

	return result, nil
}

func matchString(serializer *unSerializer) (string, error) {
	if serializer.get() != 's' {
		return "", fmt.Errorf("[%d]not a string type 's'", serializer.index)
	}
	if serializer.get() != ':' {
		return "", fmt.Errorf("[%d]not a string tag : ", serializer.index)
	}
	n, err := serializer.pickNumber(':')
	if err != nil {
		return "", err
	}
	if serializer.get() != ':' {
		return "", fmt.Errorf("[%d]not a string tag : ", serializer.index)
	}
	s, err := serializer.pickString()
	if err != nil {
		return "", err
	}

	if len(s) != n {
		return "", fmt.Errorf("[%d]not a string len %d != %d", serializer.index, len(s), n)
	}

	return s, nil
}

func matchInteger(serializer *unSerializer) (int, error) {
	if serializer.get() != 'i' {
		return 0, fmt.Errorf("[%d]not a integer type 'i'", serializer.index)
	}
	if serializer.get() != ':' {
		return 0, fmt.Errorf("[%d]not a integer tag : ", serializer.index)
	}

	n, err := serializer.pickNumber(';')
	if err != nil {
		return 0, err
	}

	return n, nil
}

func matchBool(serializer *unSerializer) (bool, error) {
	if serializer.get() != 'b' {
		return false, fmt.Errorf("[%d]not a bool type 'b'", serializer.index)
	}
	if serializer.get() != ':' {
		return false, fmt.Errorf("[%d]not a bool tag : ", serializer.index)
	}
	n, err := serializer.pickNumber(';')
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func matchNull(serializer *unSerializer) (any, error) {
	if serializer.get() != 'N' {
		return nil, fmt.Errorf("[%d]not a null type 'N'", serializer.index)
	}
	return nil, nil
}

// 解析数据
func UnSerialize(data []byte) (any, error) {
	rs := toRunes(data)
	us := newUnSerializer(rs)
	return matchType(us)
}

// ============================================================================
// 填充数据

// TODO
func fillData[T any](src any, v T) error {
	vt := reflect.TypeOf(v)
	fmt.Printf("kind: %v\n", vt.Kind())
	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}

	return nil
}

func UnSerializeTo[T any](data []byte, v T) error {
	// 解析数据
	r, err := UnSerialize(data)
	if err != nil {
		return err
	}

	// 填充数据
	return fillData(r, v)
}
