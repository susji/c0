package require

import (
	"reflect"
	"testing"

	"github.com/susji/c0/testers"
)

func Equal(t *testing.T, expect, got interface{}) {
	if !reflect.DeepEqual(expect, got) {
		testers.DumpCaller(t)
		t.Fatalf("wanted equal, but got different")
		t.Fatalf("expected: %v [%T]", expect, expect)
		t.Fatalf("got:      %v [%T]", got, got)
	}
}

func Equalf(t *testing.T, expect, got interface{}, fmt string, va ...interface{}) {
	if !reflect.DeepEqual(expect, got) {
		testers.DumpCaller(t)
		t.Fatalf(fmt, va...)
		t.Fatalf("expected: %v [%T]", expect, expect)
		t.Fatalf("got:      %v [%T]", got, got)
	}
}

func True(t *testing.T, exp bool) {
	if !exp {
		testers.DumpCaller(t)
		t.Fatal("expected true, got false")
	}
}

func Truef(t *testing.T, exp bool, fmt string, va ...interface{}) {
	if !exp {
		testers.DumpCaller(t)
		t.Fatalf(fmt, va...)
	}
}

func False(t *testing.T, exp bool) {
	if exp {
		testers.DumpCaller(t)
		t.Fatal("expected false, got true")
	}
}

func Falsef(t *testing.T, exp bool, fmt string, va ...interface{}) {
	if exp {
		testers.DumpCaller(t)
		t.Fatalf(fmt, va...)
	}
}

func Nil(t *testing.T, exp interface{}) {
	if exp != nil &&
		(reflect.ValueOf(exp).Kind() == reflect.Ptr &&
			!reflect.ValueOf(exp).IsNil()) {
		testers.DumpCaller(t)
		t.Fatalf("wanted nil, got %v of type %T", exp, exp)
	}
}

func NotNil(t *testing.T, exp interface{}) {
	if exp == nil ||
		(reflect.ValueOf(exp).Kind() == reflect.Ptr &&
			reflect.ValueOf(exp).IsNil()) {
		testers.DumpCaller(t)
		t.Fatal("wanted not nil, got nil")
	}
}
