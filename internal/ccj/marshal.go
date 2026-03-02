package ccj

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"sort"
	"strconv"
)

// Marshal 将 Go 值编码为 CCJ canonical JSON
//
// 支持的 Go 类型：
// - nil
// - bool
// - string
// - int / int8 / int16 / int32 / int64
// - uint / uint8 / uint16 / uint32 / uint64 / uintptr（必须 <= MaxInt64）
// - slice / array（注意：不支持 []byte，避免隐式规则；请在上层显式用 hex string 表示）
// - map[string]T
// - struct（读取 json tag，支持 omitempty）
//
// 禁止：float32/float64、map 非 string key、复杂不可控类型。
func Marshal(v any) ([]byte, error) {
	cv, err := fromGo(reflect.ValueOf(v))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := writeCanonical(&out, cv); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// fromGo 把 Go 值转换为 CCJ 允许的中间表示：
// nil / bool / string / int64 / []any / map[string]any
func fromGo(rv reflect.Value) (any, error) {
	if !rv.IsValid() {
		return nil, nil
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool(), nil
	case reflect.String:
		return rv.String(), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := rv.Uint()
		if u > uint64(math.MaxInt64) {
			return nil, errors.New("整数超出 int64 范围：uint 太大")
		}
		return int64(u), nil

	case reflect.Float32, reflect.Float64:
		return nil, errors.New("不允许浮点数：CCJ 只允许整数")

	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return nil, errors.New("不支持 []byte：请使用字符串字段表示")
		}
		n := rv.Len()
		arr := make([]any, 0, n)
		for i := 0; i < n; i++ {
			it, err := fromGo(rv.Index(i))
			if err != nil {
				return nil, err
			}
			arr = append(arr, it)
		}
		return arr, nil

	case reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return nil, errors.New("不支持 [N]byte：请使用字符串字段表示")
		}
		n := rv.Len()
		arr := make([]any, 0, n)
		for i := 0; i < n; i++ {
			it, err := fromGo(rv.Index(i))
			if err != nil {
				return nil, err
			}
			arr = append(arr, it)
		}
		return arr, nil

	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return nil, errors.New("map 的键必须是 string")
		}
		obj := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key().String()
			val, err := fromGo(iter.Value())
			if err != nil {
				return nil, err
			}
			obj[k] = val
		}
		return obj, nil

	case reflect.Struct:
		t := rv.Type()
		obj := make(map[string]any, t.NumField())

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // 非导出字段
				continue
			}
			name, omitempty, skip := parseJSONTag(f)
			if skip {
				continue
			}
			fv := rv.Field(i)
			if omitempty && isEmptyValue(fv) {
				continue
			}

			val, err := fromGo(fv)
			if err != nil {
				return nil, err
			}
			obj[name] = val
		}
		return obj, nil

	default:
		return nil, errors.New("不支持的 Go 类型：" + rv.Kind().String())
	}
}

func parseJSONTag(f reflect.StructField) (name string, omitempty bool, skip bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return f.Name, false, false
	}
	parts := splitComma(tag)
	name = f.Name
	if len(parts) > 0 && parts[0] != "" {
		name = parts[0]
	}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty, false
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.String:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// writeCanonical 输出 CCJ canonical JSON：
// - 无空白
// - object key 按 UTF-8 字节序升序排序
// - string 最小必要转义
func writeCanonical(w *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		w.WriteString("null")
	case bool:
		if x {
			w.WriteString("true")
		} else {
			w.WriteString("false")
		}
	case int64:
		w.WriteString(strconv.FormatInt(x, 10))
	case string:
		writeString(w, x)
	case []any:
		w.WriteByte('[')
		for i := 0; i < len(x); i++ {
			if i > 0 {
				w.WriteByte(',')
			}
			if err := writeCanonical(w, x[i]); err != nil {
				return err
			}
		}
		w.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return bytes.Compare([]byte(keys[i]), []byte(keys[j])) < 0
		})

		w.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				w.WriteByte(',')
			}
			writeString(w, k)
			w.WriteByte(':')
			if err := writeCanonical(w, x[k]); err != nil {
				return err
			}
		}
		w.WriteByte('}')
	default:
		return errors.New("写出失败：内部出现不支持的类型")
	}
	return nil
}

// writeString：最小必要转义
// - 转义 \" \\
// - 控制字符(0x00-0x1F) 必须转义：\n \r \t 或 \u00XX
func writeString(w *bytes.Buffer, s string) {
	w.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			w.WriteString(`\"`)
		case '\\':
			w.WriteString(`\\`)
		case '\n':
			w.WriteString(`\n`)
		case '\r':
			w.WriteString(`\r`)
		case '\t':
			w.WriteString(`\t`)
		default:
			if c < 0x20 {
				w.WriteString(`\u00`)
				const hex = "0123456789abcdef"
				w.WriteByte(hex[c>>4])
				w.WriteByte(hex[c&0x0f])
			} else {
				w.WriteByte(c)
			}
		}
	}
	w.WriteByte('"')
}
