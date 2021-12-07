// Copyright 2020, The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpy_test

import (
	"archive/tar"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-cpy/cpy"
)

type S struct {
	B    bool
	I    int
	I8   int8
	I16  int16
	I32  int32
	I64  int64
	U    uint
	U8   uint8
	U16  uint16
	U32  uint32
	U64  uint64
	F32  float32
	F64  float64
	C64  complex64
	C128 complex128
	S    string

	Pt  *S
	If  Proto
	Ar  [32]*M1
	Sl  []M1
	Ma1 map[string]M1
	Ma2 map[*M1]M1
	St  tar.Header
	Ch  chan int
	Fn  func() string

	Ma  M
	Mb  *M
	M1a M1
	M1b *M1
	M2a M2
	M2b *M2

	Ti  time.Time
	PTi *time.Time
}

type Proto interface{ Proto() }
type ProtoM1 interface{ ProtoM1() }
type ProtoM2 interface{ ProtoM2() }

type M struct{ A, a int }

func (*M) Proto() {}

type M1 struct{ A, a int }

func (M1) Proto()   {}
func (M1) ProtoM1() {}

type M2 struct{ A, a int }

func (*M2) Proto()   {}
func (*M2) ProtoM2() {}

var now = time.Now()

func Test(t *testing.T) {
	tests := []struct {
		src       interface{}
		cpyOpts   []cpy.Option
		cmpOpts   []cmp.Option
		verify    func(t *testing.T, dst, src interface{})
		wantPanic bool
		reason    string
	}{{
		src:     S{Ti: now},
		cmpOpts: []cmp.Option{cmpopts.IgnoreTypes(time.Time{})},
		verify: func(t *testing.T, dst, src interface{}) {
			if got := dst.(S).Ti; !got.IsZero() {
				t.Errorf("S.Ti = %v, want %v", got, time.Time{})
			}
		},
		reason: "unexported fields should be ignored",
	}, {
		src:     S{Ti: now, PTi: &now},
		cpyOpts: []cpy.Option{cpy.Shallow(time.Time{})},
		reason:  "unexported fields of time.Time copied because we permit shallow copies",
	}, {
		src:     S{Ti: now, PTi: &now},
		cpyOpts: []cpy.Option{cpy.Func(func(t time.Time) time.Time { return t })},
		reason:  "unexported fields of time.Time copied because we provide custom copy function",
	}, {
		src: S{
			B:    true,
			I:    -1,
			I8:   -8,
			I16:  -16,
			I32:  -32,
			I64:  -64,
			U:    +1,
			U8:   +8,
			U16:  +16,
			U32:  +32,
			U64:  +64,
			F32:  32.32,
			F64:  64.64,
			C64:  64i + 64,
			C128: 128i + 128,
			S:    "hello",
		},
		reason: "primitive types are shallow copied",
	}, {
		src: S{Pt: &S{S: "hello"}},
		verify: func(t *testing.T, dst, src interface{}) {
			if p1, p2 := dst.(S).Pt, src.(S).Pt; p1 == p2 {
				t.Errorf("S.Pt are equal, want inequal")
			}
		},
		reason: "pointers are deeply copied",
	}, {
		src:     S{If: &M{A: 5}},
		cpyOpts: []cpy.Option{cpy.IgnoreAllUnexported()},
		verify: func(t *testing.T, dst, src interface{}) {
			if i1, i2 := dst.(S).If, src.(S).If; i1 == i2 {
				t.Errorf("S.If are equal, want inequal")
			}
		},
		reason: "interfaces are deeply copied",
	}, {
		src:     S{Ar: [32]*M1{2: {A: 2}}},
		cpyOpts: []cpy.Option{cpy.IgnoreAllUnexported()},
		verify: func(t *testing.T, dst, src interface{}) {
			if a1, a2 := dst.(S).Ar[2], src.(S).Ar[2]; a1 == a2 {
				t.Errorf("S.Ar[2] are equal, want inequal")
			}
		},
		reason: "arrays are deeply copied",
	}, {
		src:     S{Sl: []M1{2: {A: 2}}},
		cpyOpts: []cpy.Option{cpy.IgnoreAllUnexported()},
		verify: func(t *testing.T, dst, src interface{}) {
			if a1, a2 := &dst.(S).Sl[2], &src.(S).Sl[2]; a1 == a2 {
				t.Errorf("&S.Sl[2] are equal, want inequal")
			}
		},
		reason: "slices are deeply copied",
	}, {
		src:     S{Ma1: map[string]M1{"one": {A: 1}, "two": {A: 2}}},
		cpyOpts: []cpy.Option{cpy.IgnoreAllUnexported()},
		verify: func(t *testing.T, dst, src interface{}) {
			if m1, m2 := dst.(S).Ma1, src.(S).Ma1; reflect.ValueOf(m1).Pointer() == reflect.ValueOf(m2).Pointer() {
				t.Errorf("S.Ma1 are equal, want inequal")
			}
		},
		reason: "maps are deeply copied",
	}, {
		src:     S{Ma2: map[*M1]M1{{A: 1}: {A: 1}, {A: 2}: {A: 2}}},
		cpyOpts: []cpy.Option{cpy.IgnoreAllUnexported()},
		cmpOpts: []cmp.Option{cmpopts.SortMaps(func(x, y *M1) bool {
			return x.A < y.A
		})},
		verify: func(t *testing.T, dst, src interface{}) {
			m1, m2 := dst.(S).Ma2, src.(S).Ma2
			if reflect.ValueOf(m1).Pointer() == reflect.ValueOf(m2).Pointer() {
				t.Errorf("S.Ma2 are equal, want inequal")
			}
			for k := range m1 {
				if _, ok := m2[k]; ok {
					t.Errorf("S.Ma2[%p] found in both maps, want not found", k)
				}
			}
		},
		reason: "maps are deeply copied, but keys are shallow copied",
	}, {
		src:     S{St: tar.Header{PAXRecords: map[string]string{"key": "value"}}},
		cpyOpts: []cpy.Option{cpy.IgnoreAllUnexported()},
		verify: func(t *testing.T, dst, src interface{}) {
			if m1, m2 := dst.(S).St.PAXRecords, src.(S).St.PAXRecords; reflect.ValueOf(m1).Pointer() == reflect.ValueOf(m2).Pointer() {
				t.Errorf("S.St.PAXRecords are equal, want inequal")
			}
		},
		reason: "structs are deeply copied, but keys are shallow copied",
	}, {
		src:     S{Ch: make(chan int), Fn: func() string { return "myfunc" }},
		cmpOpts: []cmp.Option{cmpopts.IgnoreTypes((func() string)(nil))},
		verify: func(t *testing.T, dst, src interface{}) {
			if f := dst.(S).Fn; f() != "myfunc" {
				t.Errorf("S.Fn = %p, want %p", f, src.(S).Fn)
			}
		},
		reason: "channels and functions are shallow copied",
	}, {
		src: S{Ma: M{a: 1}, Mb: &M{a: 2}},
		cpyOpts: []cpy.Option{
			cpy.Func(func(m M) M { return M{A: m.A, a: m.a} }),
		},
		reason: "copy function on value receiver is used",
	}, {
		src: S{Ma: M{a: 1}, Mb: &M{a: 2}},
		cpyOpts: []cpy.Option{
			cpy.Func(func(m *M) *M { return &M{A: m.A, a: m.a} }),
		},
		reason: "copy function on pointer receiver is used",
	}, {
		src: S{Ma: M{a: 1}, Mb: &M{a: 2}},
		cpyOpts: []cpy.Option{
			cpy.Func(func(m *M) *M { panic("want not called") }),
			cpy.Func(func(m *M) *M { return &M{A: m.A, a: m.a} }),
		},
		reason: "first copy function not used given a latter copy function",
	}, {
		src: S{Ma: M{a: 1}, Mb: &M{a: 2}},
		cpyOpts: []cpy.Option{
			cpy.Func(func(Proto) Proto { panic("want not called") }),
			cpy.Func(func(m *M) *M { return &M{A: m.A, a: m.a} }),
		},
		reason: "interface copy function not used given a concrete copy function",
	}, {
		src: S{Ma: M{a: 1}, Mb: &M{a: 2}},
		cpyOpts: []cpy.Option{
			cpy.Func(func(m M) M { return M{A: m.A, a: m.a} }),
		},
		reason: "copy function on value receiver is used",
	}, {
		src: S{Ma: M{a: 1}, Mb: &M{a: 2}},
		cpyOpts: []cpy.Option{
			cpy.Func(func(m *M) *M { return &M{A: m.A, a: m.a} }),
		},
		reason: "copy function on pointer receiver is used",
	}, {
		src: S{
			If: &M{a: 100},
			Ma: M{a: 1}, Mb: &M{a: 2},
			M1a: M1{a: 1}, M1b: &M1{a: 2},
			M2a: M2{a: 1}, M2b: &M2{a: 2},
		},
		cpyOpts: []cpy.Option{
			cpy.Func(func(m Proto) Proto { panic("want not called") }),
			cpy.Func(func(m Proto) Proto {
				switch m := m.(type) {
				case *M:
					return &M{A: m.A, a: m.a}
				case M1:
					return M1{A: m.A, a: m.a}
				case *M2:
					return &M2{A: m.A, a: m.a}
				default:
					panic(fmt.Sprintf("unexpected type %T", m))
				}
			}),
		},
		reason: "*M, M1, *M2 all implement Proto; copy function on interface should handle all three",
	}, {
		src: S{
			M1a: M1{a: 1}, M1b: &M1{a: 2},
			M2a: M2{a: 1}, M2b: &M2{a: 2},
		},
		cpyOpts: []cpy.Option{
			cpy.Func(func(m Proto) Proto { panic("want not called") }),
			cpy.Func(func(m ProtoM1) ProtoM1 {
				return M1{A: m.(M1).A, a: m.(M1).a}
			}),
			cpy.Func(func(m ProtoM2) ProtoM2 {
				return &M2{A: m.(*M2).A, a: m.(*M2).a}
			}),
		},
		reason: "copy function on Proto not called given copy functions on ProtoM1 and ProtoM2",
	}, {
		src: S{
			Ma: M{a: 1},
		},
		cpyOpts: []cpy.Option{
			cpy.Func(func(m M) M { return M{A: m.A, a: m.a} }),
			cpy.Func(func(m Proto) Proto { panic("want not called") }),
		},
		reason: "copy function on concrete type takes precedence over interface type",
	}}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			defer func() {
				gotPanic := recover() != nil
				if gotPanic != tt.wantPanic {
					t.Errorf("got panic=%v, want panic=%v", gotPanic, tt.wantPanic)
				}
			}()

			tt.cpyOpts = append(tt.cpyOpts, cpy.IgnoreAllUnexported())
			copier := cpy.New(tt.cpyOpts...)
			dst := copier.Copy(tt.src)
			tt.cmpOpts = append(tt.cmpOpts, cmp.AllowUnexported(S{}, M{}, M1{}, M2{}))
			if diff := cmp.Diff(dst, tt.src, tt.cmpOpts...); diff != "" {
				t.Errorf("Copy() mismatch (-want +got):\n%s", diff)
			}
			if tt.verify != nil {
				tt.verify(t, dst, tt.src)
			}
		})
	}
}
