package ccj

import (
	"bytes"
	"testing"
)

// -------------------- 测试用结构体 --------------------

type Inner struct {
	Note string  `json:"note"`
	Data []byte  `json:"data"` // []byte -> hex string
	Sig  [4]byte `json:"sig"`  // [N]byte -> hex string
}

type Outer struct {
	ChainID string `json:"chainId"`
	Inner   Inner  `json:"inner"`

	// map[[N]byte] 的 key 会被编码为 hex string（JSON object key）
	Items map[[4]byte]Inner `json:"items"`
}

// -------------------- 辅助函数 --------------------

func mustKey4(b0, b1, b2, b3 byte) [4]byte { return [4]byte{b0, b1, b2, b3} }

func contains(s, sub string) bool { return bytes.Contains([]byte(s), []byte(sub)) }

// -------------------- 正例：嵌套 struct + bytes roundtrip --------------------

func TestMarshalUnmarshal_NestedStruct_WithBytes(t *testing.T) {
	in := Outer{
		ChainID: "csc2-dev",
		Inner: Inner{
			Note: "hello",
			Data: []byte{0x00, 0x01, 0x2a, 0xff},
			Sig:  [4]byte{0xde, 0xad, 0xbe, 0xef},
		},
		Items: map[[4]byte]Inner{
			mustKey4(0x01, 0x02, 0x03, 0x04): {
				Note: "item1",
				Data: []byte{0xaa, 0xbb},
				Sig:  [4]byte{0x00, 0x00, 0x00, 0x01},
			},
		},
	}

	b, err := Marshal(in)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 检查 bytes 是否被编码为小写 hex string（canonical）
	s := string(b)
	if !contains(s, `"data":"00012aff"`) {
		t.Fatalf("expected inner.data hex, got: %s", s)
	}
	if !contains(s, `"sig":"deadbeef"`) {
		t.Fatalf("expected inner.sig hex, got: %s", s)
	}

	// map[[4]byte] 的 key 应编码成 hex string "01020304"
	if !contains(s, `"01020304"`) {
		t.Fatalf("expected map key hex 01020304, got: %s", s)
	}
	// value 内 bytes 也应 hex
	if !contains(s, `"data":"aabb"`) {
		t.Fatalf("expected item1.data hex aabb, got: %s", s)
	}

	var out Outer
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Roundtrip 检查
	if out.ChainID != in.ChainID {
		t.Fatalf("chainId mismatch: %q != %q", out.ChainID, in.ChainID)
	}
	if out.Inner.Note != in.Inner.Note {
		t.Fatalf("inner.note mismatch: %q != %q", out.Inner.Note, in.Inner.Note)
	}
	if !bytes.Equal(out.Inner.Data, in.Inner.Data) {
		t.Fatalf("inner.data mismatch: %x != %x", out.Inner.Data, in.Inner.Data)
	}
	if out.Inner.Sig != in.Inner.Sig {
		t.Fatalf("inner.sig mismatch: %x != %x", out.Inner.Sig, in.Inner.Sig)
	}

	if len(out.Items) != len(in.Items) {
		t.Fatalf("items len mismatch: %d != %d", len(out.Items), len(in.Items))
	}
	k := mustKey4(0x01, 0x02, 0x03, 0x04)
	got, ok := out.Items[k]
	if !ok {
		t.Fatalf("items missing key %x", k)
	}
	exp := in.Items[k]
	if got.Note != exp.Note || !bytes.Equal(got.Data, exp.Data) || got.Sig != exp.Sig {
		t.Fatalf("items[%x] mismatch: got=%+v exp=%+v", k, got, exp)
	}
}

// -------------------- 正例：map[[N]byte] key 多个值 + 关键排序不要求一致（只要求可逆） --------------------

func TestMarshalUnmarshal_MapKeyBytes_Multiple(t *testing.T) {
	type V struct {
		X uint64 `json:"x"`
		B []byte `json:"b"`
	}
	type M struct {
		Map map[[2]byte]V `json:"map"`
	}

	in := M{
		Map: map[[2]byte]V{
			{0x00, 0x01}: {X: 1, B: []byte{0x10}},
			{0x00, 0x02}: {X: 2, B: []byte{0x20, 0x21}},
		},
	}

	b, err := Marshal(in)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	s := string(b)

	// key 应出现为 "0001" / "0002"
	if !contains(s, `"0001"`) || !contains(s, `"0002"`) {
		t.Fatalf("expected map keys 0001/0002, got: %s", s)
	}

	var out M
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(out.Map) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(out.Map))
	}
	if out.Map[[2]byte{0x00, 0x01}].X != 1 || !bytes.Equal(out.Map[[2]byte{0x00, 0x01}].B, []byte{0x10}) {
		t.Fatalf("entry 0001 mismatch: %+v", out.Map[[2]byte{0x00, 0x01}])
	}
	if out.Map[[2]byte{0x00, 0x02}].X != 2 || !bytes.Equal(out.Map[[2]byte{0x00, 0x02}].B, []byte{0x20, 0x21}) {
		t.Fatalf("entry 0002 mismatch: %+v", out.Map[[2]byte{0x00, 0x02}])
	}
}

// -------------------- 负例：大写 hex 应拒绝（严格小写） --------------------

func TestUnmarshal_RejectUpperHex_ForBytesField(t *testing.T) {
	type X struct {
		Data []byte `json:"data"`
	}
	j := `{"data":"AA"}`
	var x X
	if err := Unmarshal([]byte(j), &x); err == nil {
		t.Fatalf("expected error for upper hex, got nil")
	}
}

// map[[N]byte] 的 key 也应拒绝大写
func TestUnmarshal_RejectUpperHex_ForMapKey(t *testing.T) {
	type X struct {
		M map[[2]byte]uint64 `json:"m"`
	}
	j := `{"m":{"AA00":1}}`
	var x X
	if err := Unmarshal([]byte(j), &x); err == nil {
		t.Fatalf("expected error for upper hex map key, got nil")
	}
}

// -------------------- 负例：奇数长度 hex 应拒绝 --------------------

func TestUnmarshal_RejectOddLengthHex(t *testing.T) {
	type X struct {
		Data []byte `json:"data"`
	}
	j := `{"data":"abc"}`
	var x X
	if err := Unmarshal([]byte(j), &x); err == nil {
		t.Fatalf("expected error for odd-length hex, got nil")
	}
}

// -------------------- 负例：map key 长度与 [N]byte 不匹配应拒绝 --------------------

func TestUnmarshal_RejectMapKeyLengthMismatch(t *testing.T) {
	type X struct {
		M map[[4]byte]uint64 `json:"m"`
	}
	// key "aa" 解码后只有 1 byte，但目标 key 是 [4]byte
	j := `{"m":{"aa":1}}`
	var x X
	if err := Unmarshal([]byte(j), &x); err == nil {
		t.Fatalf("expected error for map key length mismatch, got nil")
	}
}
