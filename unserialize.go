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
	case 'd':
		return matchFloat(serializer)
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

		// value
		value, err := matchType(serializer)
		if err != nil {
			return nil, err
		}

		result[key] = value
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

		// value
		value, err := matchType(serializer)
		if err != nil {
			return nil, err
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

		// value
		value, err := matchType(serializer)
		if err != nil {
			return nil, err
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

	if serializer.get() != ';' {
		return "", fmt.Errorf("[%d]not a string tag end ;", serializer.index)
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

	if serializer.get() != ';' {
		return 0, fmt.Errorf("[%d]not a integer tag end ;", serializer.index)
	}

	return n, nil
}

func matchFloat(serializer *unSerializer) (float64, error) {
	if serializer.get() != 'd' {
		return 0, fmt.Errorf("[%d]not a float type 'd'", serializer.index)
	}
	if serializer.get() != ':' {
		return 0, fmt.Errorf("[%d]not a float tag : ", serializer.index)
	}
	h, err := serializer.pickNumber('.')
	if err != nil {
		return 0, err
	}
	if serializer.get() != '.' {
		return 0, fmt.Errorf("[%d]not a float tag . ", serializer.index)
	}
	t, err := serializer.pickNumber(';')
	if err != nil {
		return 0, err
	}
	if serializer.get() != ';' {
		return 0, fmt.Errorf("[%d]not a float tag ; ", serializer.index)
	}
	return strconv.ParseFloat(fmt.Sprintf("%d.%d", h, t), 64)
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

	if serializer.get() != ';' {
		return 0, fmt.Errorf("[%d]not a null end tag ; ", serializer.index)
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

func fillMap(m map[string]any, fv reflect.Value) error {
	rm := reflect.MakeMap(reflect.TypeOf(m))
	for k, v := range m {
		rm.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
	}
	fv.Set(rm)
	return nil
}

func fillStruct(src map[string]any, vt reflect.Type, vv reflect.Value) error {
	// 遍历字段
	for i := 0; i < vt.NumField(); i++ {
		ft := vt.Field(i)
		fv := vv.Field(i)

		fn := ft.Tag.Get("php")
		if v, ok := src[fn]; ok {
			fk := ft.Type.Kind()
			switch fk {
			case reflect.Int:
				if i, ok := v.(int); ok {
					fv.SetInt(int64(i))
				}
			case reflect.Float64:
				if f, ok := v.(float64); ok {
					fv.SetFloat(f)
				}
			case reflect.String:
				if s, ok := v.(string); ok {
					fv.SetString(s)
				}
			case reflect.Slice:
				if s, ok := v.([]any); ok {
					rs := reflect.MakeSlice(ft.Type, len(s), len(s))
					for i := 0; i < len(s); i++ {
						rs.Index(i).Set(reflect.ValueOf(s[i]))
					}
					fv.Set(rs)
					// fmt.Printf("slice %v\n", s)
				}
			case reflect.Map:
				if m, ok := v.(map[string]any); ok {
					if err := fillMap(m, fv); err != nil {
						return err
					}
				}
			case reflect.Struct:
				if m, ok := v.(map[string]any); ok {
					if err := fillStruct(m, ft.Type, fv); err != nil {
						return err
					}
				}
			}
			// fmt.Printf("fk: %v %v\n", fk, ft)
		}
	}
	return nil
}

func fillData[T any](src map[string]any, v T) error {
	vt := reflect.TypeOf(v)
	if vt.Kind() != reflect.Ptr {
		return fmt.Errorf("root target value must a pointer '%s'", vt.Name())
	}
	vt = vt.Elem()
	if vt.Kind() != reflect.Struct {
		return fmt.Errorf("root target value must a struct pointer '%s'", vt.Name())
	}

	return fillStruct(src, vt, reflect.ValueOf(v).Elem())
}

func UnSerializeTo[T any](data []byte, v T) error {
	// 解析数据
	r, err := UnSerialize(data)
	if err != nil {
		return err
	}
	m, ok := r.(map[string]any)
	if !ok {
		return fmt.Errorf("UnSerialize result must a map[string]any")
	}

	// 填充数据
	return fillData(m, v)
}
