package ccj

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"CrowdShardChain/internal/hash"
)

// Unmarshal 将 CCJ JSON 自动注入到 out。
func Unmarshal(data []byte, out any) error {
	if out == nil {
		return errors.New("Unmarshal 失败：out 不能为空")
	}
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("Unmarshal 失败：out 必须是非空指针")
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	v, err := readValueStrict(dec)
	if err != nil {
		return err
	}
	if err := ensureEOF(dec); err != nil {
		return err
	}
	return inject(rv.Elem(), v)
}

func ensureEOF(dec *json.Decoder) error {
	_, err := dec.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return errors.New("JSON 末尾存在多余内容")
	}
	return errors.New("JSON 末尾存在多余内容")
}

func readValueStrict(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, errors.New("JSON 解析失败：" + err.Error())
	}

	switch t := tok.(type) {
	case json.Delim:
		switch byte(t) {
		case '{':
			return readObjectStrict(dec)
		case '[':
			return readArrayStrict(dec)
		default:
			return nil, errors.New("JSON 语法错误：未知分隔符")
		}
	case string:
		return t, nil
	case bool:
		return t, nil
	case nil:
		return nil, nil
	case json.Number:
		i64, err := parseStrictInt64(t.String())
		if err != nil {
			return nil, err
		}
		return i64, nil
	case float64:
		return nil, errors.New("不允许浮点数：CCJ 只允许整数")
	default:
		return nil, errors.New("不支持的 JSON 类型")
	}
}

func readArrayStrict(dec *json.Decoder) (any, error) {
	var arr []any
	for dec.More() {
		v, err := readValueStrict(dec)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, errors.New("JSON 数组解析失败：" + err.Error())
	}
	if d, ok := tok.(json.Delim); !ok || byte(d) != ']' {
		return nil, errors.New("JSON 语法错误：数组未正确结束")
	}
	return arr, nil
}

func readObjectStrict(dec *json.Decoder) (any, error) {
	obj := make(map[string]any, 8)
	seen := make(map[string]struct{}, 8)

	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, errors.New("JSON 对象解析失败：" + err.Error())
		}
		key, ok := kt.(string)
		if !ok {
			return nil, errors.New("JSON 语法错误：对象键必须是字符串")
		}
		if _, exists := seen[key]; exists {
			return nil, errors.New("JSON 非法：对象存在重复键：" + key)
		}
		seen[key] = struct{}{}

		val, err := readValueStrict(dec)
		if err != nil {
			return nil, err
		}
		obj[key] = val
	}

	tok, err := dec.Token()
	if err != nil {
		return nil, errors.New("JSON 对象解析失败：" + err.Error())
	}
	if d, ok := tok.(json.Delim); !ok || byte(d) != '}' {
		return nil, errors.New("JSON 语法错误：对象未正确结束")
	}
	return obj, nil
}

// parseStrictInt64：只允许十进制整数（禁止小数/科学计数/前导+/-0/前导零）
func parseStrictInt64(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("整数格式错误：为空")
	}
	for _, c := range s {
		if c == '.' || c == 'e' || c == 'E' {
			return 0, errors.New("不允许浮点或科学计数法：" + s)
		}
	}
	if s[0] == '+' {
		return 0, errors.New("整数格式错误：不允许前导 +：" + s)
	}
	if s == "-0" {
		return 0, errors.New("整数格式错误：不允许 -0")
	}
	if s[0] == '0' && len(s) > 1 {
		return 0, errors.New("整数格式错误：不允许前导 0：" + s)
	}
	if s[0] == '-' {
		if len(s) == 1 {
			return 0, errors.New("整数格式错误：仅有 '-'")
		}
		if s[1] == '0' && len(s) > 2 {
			return 0, errors.New("整数格式错误：不允许 -012 形式：" + s)
		}
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, errors.New("整数超出 int64 范围或格式错误：" + s)
	}
	return v, nil
}

// ---------------- 注入：中间值 -> 目标对象（反射） ----------------

func inject(dst reflect.Value, v any) error {
	// 指针：必要时分配
	if dst.Kind() == reflect.Pointer {
		if v == nil {
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		}
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return inject(dst.Elem(), v)
	}

	// v 为 null
	if v == nil {
		switch dst.Kind() {
		case reflect.Interface, reflect.Slice, reflect.Map:
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		default:
			return errors.New("类型不匹配：不能将 null 注入到 " + dst.Type().String())
		}
	}

	switch dst.Kind() {
	case reflect.Bool:
		b, ok := v.(bool)
		if !ok {
			return typeMismatch(dst, v)
		}
		dst.SetBool(b)
		return nil

	case reflect.String:
		s, ok := v.(string)
		if !ok {
			return typeMismatch(dst, v)
		}
		dst.SetString(s)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, ok := v.(int64)
		if !ok {
			return typeMismatch(dst, v)
		}
		if dst.OverflowInt(i) {
			return errors.New("整数溢出：无法注入到 " + dst.Type().String())
		}
		dst.SetInt(i)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		i, ok := v.(int64)
		if !ok {
			return typeMismatch(dst, v)
		}
		if i < 0 {
			return errors.New("类型不匹配：不能将负整数注入到无符号类型 " + dst.Type().String())
		}
		u := uint64(i)
		if dst.OverflowUint(u) {
			return errors.New("无符号整数溢出：无法注入到 " + dst.Type().String())
		}
		dst.SetUint(u)
		return nil

	case reflect.Slice:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			s, ok := v.(string)
			if !ok {
				return typeMismatch(dst, v)
			}
			b, err := hash.HexToBytes(s)
			if err != nil {
				return err
			}
			out := reflect.MakeSlice(dst.Type(), len(b), len(b))
			reflect.Copy(out, reflect.ValueOf(b))
			dst.Set(out)
			return nil
		}

		arr, ok := v.([]any)
		if !ok {
			return typeMismatch(dst, v)
		}
		out := reflect.MakeSlice(dst.Type(), len(arr), len(arr))
		for i := 0; i < len(arr); i++ {
			if err := inject(out.Index(i), arr[i]); err != nil {
				return fmt.Errorf("数组元素注入失败 index=%d：%w", i, err)
			}
		}
		dst.Set(out)
		return nil

	case reflect.Array:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			s, ok := v.(string)
			if !ok {
				return typeMismatch(dst, v)
			}
			b, err := hash.HexToBytes(s)
			if err != nil {
				return err
			}
			if len(b) != dst.Len() {
				return fmt.Errorf("byte array 长度不匹配：目标=%d 输入=%d", dst.Len(), len(b))
			}
			for i := 0; i < dst.Len(); i++ {
				dst.Index(i).SetUint(uint64(b[i]))
			}
			return nil
		}

		arr, ok := v.([]any)
		if !ok {
			return typeMismatch(dst, v)
		}
		if len(arr) != dst.Len() {
			return fmt.Errorf("数组长度不匹配：目标=%d 输入=%d", dst.Len(), len(arr))
		}
		for i := 0; i < len(arr); i++ {
			if err := inject(dst.Index(i), arr[i]); err != nil {
				return fmt.Errorf("数组元素注入失败 index=%d：%w", i, err)
			}
		}
		return nil

	case reflect.Map:
		obj, ok := v.(map[string]any)
		if !ok {
			return typeMismatch(dst, v)
		}

		kt := dst.Type().Key()
		et := dst.Type().Elem()

		// 1) map[string]T（原样）
		if kt.Kind() == reflect.String {
			if dst.IsNil() {
				dst.Set(reflect.MakeMapWithSize(dst.Type(), len(obj)))
			}
			for k, vv := range obj {
				ev := reflect.New(et).Elem()
				if err := inject(ev, vv); err != nil {
					return fmt.Errorf("map 值注入失败 key=%s：%w", k, err)
				}
				dst.SetMapIndex(reflect.ValueOf(k), ev)
			}
			return nil
		}

		// 2) map[[N]byte]T（新增）
		if !(kt.Kind() == reflect.Array && kt.Elem().Kind() == reflect.Uint8) {
			return errors.New("map key 必须是 string 或 [N]byte：" + dst.Type().String())
		}

		if dst.IsNil() {
			dst.Set(reflect.MakeMapWithSize(dst.Type(), len(obj)))
		}

		// 将 obj 的 string key（hex）解码到 [N]byte
		N := kt.Len()
		for kHex, vv := range obj {
			kb, err := hash.HexToBytes(kHex) // 严格小写 hex 解码
			if err != nil {
				return fmt.Errorf("map key hex 非法 key=%s：%w", kHex, err)
			}
			if len(kb) != N {
				return fmt.Errorf("map key 长度不匹配：目标=%d 输入=%d key=%s", N, len(kb), kHex)
			}

			// 构造 [N]byte 的 reflect.Value
			keyArr := reflect.New(kt).Elem()
			for i := 0; i < N; i++ {
				keyArr.Index(i).SetUint(uint64(kb[i]))
			}

			ev := reflect.New(et).Elem()
			if err := inject(ev, vv); err != nil {
				return fmt.Errorf("map 值注入失败 key=%s：%w", kHex, err)
			}
			dst.SetMapIndex(keyArr, ev)
		}
		return nil

	case reflect.Struct:
		obj, ok := v.(map[string]any)
		if !ok {
			return typeMismatch(dst, v)
		}
		return injectStruct(dst, obj)

	case reflect.Interface:
		dst.Set(reflect.ValueOf(v))
		return nil

	default:
		return errors.New("不支持注入到目标类型：" + dst.Type().String())
	}
}

func injectStruct(dst reflect.Value, obj map[string]any) error {
	t := dst.Type()

	nameToIndex := make(map[string]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name, _, skip := parseJSONTag(f)
		if skip {
			continue
		}
		nameToIndex[name] = i
	}

	for k, vv := range obj {
		fi, ok := nameToIndex[k]
		if !ok {
			continue // 未知字段忽略
		}
		fv := dst.Field(fi)
		if !fv.CanSet() {
			continue
		}
		if err := inject(fv, vv); err != nil {
			return fmt.Errorf("字段注入失败 %s：%w", k, err)
		}
	}
	return nil
}

func typeMismatch(dst reflect.Value, v any) error {
	return fmt.Errorf("类型不匹配：目标=%s 输入类型=%T", dst.Type().String(), v)
}
