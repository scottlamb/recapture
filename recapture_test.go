package recapture

import (
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func Test_ArgCountMismatch(t *testing.T) {
	var panicReason interface{}
	r := regexp.MustCompile("^nada$")
	{
		defer func() { panicReason = recover() }()
	}
	var extraArg int
	MatchString(r, "nada", &extraArg)
	if panicReason != "Expected 0 arguments, got 1" {
		t.Errorf("panicReason = %v", panicReason)
	}
}

func Test_SaveString(t *testing.T) {
	var s string
	err := save("foo", &s)
	if err != nil {
		t.Errorf("err = %v", err)
	}
	if s != "foo" {
		t.Errorf("s = %v", s)
	}
}

func Test_SaveIntegerSuccess(t *testing.T) {
	var i int
	err := save("42", &i)
	if err != nil {
		t.Errorf("err = %v", err)
	}
	if i != 42 {
		t.Errorf("i = %v", i)
	}
}

func Test_SaveHexSuccess(t *testing.T) {
	var i int64
	err := save("deadbeef", Hex(&i))
	if err != nil {
		t.Errorf("err = %v", err)
	}
	if i != 0xdeadbeef {
		t.Errorf("i = %v", i)
	}
}

func Test_SaveIntegerFailure(t *testing.T) {
	var i int
	err := save("asdf", &i)
	if err == nil {
		t.Error("no error")
	}
}

func Test_SaveFmtSuccess(t *testing.T) {
	var i int
	err := save("010", Fmt("%v", &i))
	if err != nil {
		t.Errorf("err = %v", err)
	}
	if i != 010 {
		t.Errorf("i = %d", i)
	}
}

func Test_SaveFmtFailure(t *testing.T) {
	var i int
	err := save("asdf", Fmt("%d", &i))
	if err == nil {
		t.Error("missing err")
	}
}

func Test_SaveFmtPartial(t *testing.T) {
	var i int
	err := save("010asdf", Fmt("%v", &i))
	if err.Error() != "did not consume last 4 bytes of input 010asdf" {
		t.Errorf("err = %v", err)
	}
}

func Test_SaveByteSuccess(t *testing.T) {
	var b byte
	err := save("x", Byte(&b))
	if err != nil {
		t.Errorf("err = %v", err)
	}
	if b != 'x' {
		t.Errorf("b = %v", b)
	}
}

func Test_SaveByteExtra(t *testing.T) {
	var b byte
	err := save("foo", Byte(&b))
	if err == nil || err.Error() != "expected 1 byte, got 3: foo" {
		t.Errorf("err = %v", err)
	}
}

func Test_SaveRuneSuccess(t *testing.T) {
	var r rune
	err := save("\u2026", Rune(&r))
	if err != nil {
		t.Errorf("err = %v", err)
	}
	if r != '\u2026' {
		t.Errorf("r = %v", r)
	}
}

func Test_SaveRuneExtra(t *testing.T) {
	input := "\u2026\u2026"
	var r rune
	err := save("\u2026\u2026", Rune(&r))
	if err == nil || err.Error() != "did not consume last 3 bytes of "+input {
		t.Errorf("err = %v", err)
	}
}

func Test_SaveRuneEmpty(t *testing.T) {
	var r rune
	err := save("", Rune(&r))
	if err != io.EOF {
		t.Errorf("err = %v", err)
	}
}

// Seems odd, but partial runes seem to be treated like full ones.
// This test is disabled for now.
func xTest_SaveRunePartial(t *testing.T) {
	input := string([]byte{'\xe2'})
	var r rune
	err := save(input, Rune(&r))
	t.Errorf("err = %v", err)
}

// A full test of matching a date string.
// Note that "09" is chosen to test that we're using the correct base.
func Test_MatchDate(t *testing.T) {
	r := regexp.MustCompile("^([0-9]{4})-([0-9]{2})-([0-9]{2})$")
	var m1, m2, m3 int
	err := MatchString(r, "2013-09-26", &m1, &m2, &m3)
	if err != nil {
		t.Errorf("err = %v", err)
	}
	if m1 != 2013 {
		t.Errorf("m1 = %v", m1)
	}
	if m2 != 9 {
		t.Errorf("m2 = %v", m2)
	}
	if m3 != 26 {
		t.Errorf("m3 = %v", m3)
	}
}

// A failed match should give a diagnostic with the original input.
func Test_MatchFailure(t *testing.T) {
	r := regexp.MustCompile(
		"^([BG]) ([0-9]+)'([0-9]+)\" ([0-9]+)'([0-9]+)\"$")
	input := "B 5'11\" 6'2"
	var m1 string
	var m2, m3, m4, m5 int
	err := MatchString(r, input, &m1, &m2, &m3, &m4, &m5)
	if err == nil || !strings.Contains(err.Error(), strconv.Quote(input)) {
		t.Errorf("err: %v", err)
	}
}

// A failed save should give diagnostics as well.
func Test_MatchSaveFailure(t *testing.T) {
	r := regexp.MustCompile(`(.*)`)
	input := "asdf"
	var m1 int
	err := MatchString(r, input, &m1)
	if err == nil || !strings.Contains(err.Error(), strconv.Quote(input)) {
		t.Errorf("err: %v", err)
	}
}
