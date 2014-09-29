// Package recapture is a helper for capturing regular expressions.
// It supports matching regular expressions and saving subcapture
// results directly into various kinds of objects. It is modeled
// after the RE2 C++ API.
//
// Example:
//
// 	r := regexp.MustCompile("(.*) (.*) (.*) (.*)")
// 	var a, b, c, d int
// 	err := recapture.MatchString(
// 		r, "100 40 0100 0x40",
// 		recapture.Octal(&a), recapture.Hex(&b),
// 		recapture.CRadix(&c), recapture.CRadix(&d))
// 	if err == nil {
// 		// prints 64 64 64 64
// 		fmt.Printf("%d %d %d %d", a, b, c, d)
// 	} else {
// 		fmt.Printf("match failed: %v", err)
// 	}
//
package recapture

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Saver saves a match as a side effect or returns error.
type Saver interface {
	Save(submatch string) error
}

type fmtarg struct {
	format string
	args   []interface{}
}

// Fmt returns a Saver that will interpret strings as with fmt.Scanf.
// It fails if the format string does not fully consume the submatch.
func Fmt(format string, args ...interface{}) *fmtarg {
	return &fmtarg{format, args}
}

func (f *fmtarg) Save(submatch string) error {
	reader := strings.NewReader(submatch)
	_, err := fmt.Fscanf(reader, f.format, f.args...)
	switch {
	case err != nil:
		return err
	case reader.Len() > 0:
		return fmt.Errorf(
			"did not consume last %d bytes of input %v",
			reader.Len(), submatch)
	}
	return nil
}

type integerSaver struct {
	radix int
	arg   interface{}
}

// Hex returns a Saver that will interpret base-16 integers, saving the result
// to the location pointed to by 'arg'.
func Hex(arg interface{}) integerSaver {
	return integerSaver{16, arg}
}

// Octal returns a Saver that will interpret base-8 integers, saving the
// result to the location pointed to by 'arg'.
func Octal(arg interface{}) integerSaver {
	return integerSaver{8, arg}
}

// CRadix returns a Saver that will intepret numbers as in the C programming
// language. It defaults to base-10 but understands the prefix "0" to mean
// base-8 and the prefix "0x" to mean base-16. It saves the result to
// the location pointed to by 'arg'.
func CRadix(arg interface{}) integerSaver {
	return integerSaver{0, arg}
}

type runeSaver struct{ *rune }

// Rune returns a Saver that saves a single rune to a location pointed to by
// 'r'.
func Rune(r *rune) runeSaver {
	return runeSaver{r}
}

func (r runeSaver) Save(submatch string) (err error) {
	reader := strings.NewReader(submatch)
	*r.rune, _, err = reader.ReadRune()
	if err == nil && reader.Len() > 0 {
		err = fmt.Errorf("did not consume last %d bytes of %v", reader.Len(), submatch)
	}
	return
}

type byteSaver struct{ *byte }

// Byte returns a Saver that saves a single byte to a location pointed to by 'b'.
func Byte(b *byte) byteSaver {
	return byteSaver{b}
}

func (b byteSaver) Save(submatch string) (err error) {
	if len(submatch) != 1 {
		return fmt.Errorf("expected 1 byte, got %d: %v", len(submatch), submatch)
	}
	*b.byte = submatch[0]
	return nil
}

func (i integerSaver) Save(submatch string) (err error) {
	switch arg := i.arg.(type) {
	case *int:
		var t int64
		t, err = strconv.ParseInt(submatch, i.radix, 0)
		*arg = int(t)
	case *uint:
		var t uint64
		t, err = strconv.ParseUint(submatch, i.radix, 0)
		*arg = uint(t)
	case *int8:
		var t int64
		t, err = strconv.ParseInt(submatch, i.radix, 8)
		*arg = int8(t)
	case *uint8:
		var t uint64
		t, err = strconv.ParseUint(submatch, i.radix, 8)
		*arg = uint8(t)
	case *int16:
		var t int64
		t, err = strconv.ParseInt(submatch, i.radix, 16)
		*arg = int16(t)
	case *uint16:
		var t uint64
		t, err = strconv.ParseUint(submatch, i.radix, 16)
		*arg = uint16(t)
	case *int32:
		var t int64
		t, err = strconv.ParseInt(submatch, i.radix, 32)
		*arg = int32(t)
	case *uint32:
		var t uint64
		t, err = strconv.ParseUint(submatch, i.radix, 32)
		*arg = uint32(t)
	case *int64:
		*arg, err = strconv.ParseInt(submatch, i.radix, 64)
	case *uint64:
		*arg, err = strconv.ParseUint(submatch, i.radix, 64)
	default:
		panic(fmt.Sprintf("Unknown number type %T", arg))
	}
	return
}

func save(submatch string, arg interface{}) (err error) {
	switch arg := arg.(type) {
	case Saver:
		err = arg.Save(submatch)
	case *int, *uint, *int8, *uint8, *int16, *uint16, *int32, *uint32, *int64, *uint64:
		return integerSaver{10, arg}.Save(submatch)
	case *bool:
		*arg, err = strconv.ParseBool(submatch)
	case *float32:
		var f float64
		f, err = strconv.ParseFloat(submatch, 32)
		*arg = float32(f)
	case *float64:
		*arg, err = strconv.ParseFloat(submatch, 64)
	case *string:
		*arg = submatch
	default:
		panic(fmt.Sprintf("Unknown argument type %T", arg))
	}
	return
}

// MatchString matches a string against a regular expression, capturing
// numbered submatches into the corresponding positional arguments.
//
// 'args' may be string pointers, number pointers, boolean pointers, or
// Savers. Note that byte/uint8 and rune/int32 are considered numbers,
// not bytes/runes. (Use the Byte/Rune savers if byte/rune behavior is
// desired.) complex64 and complex128 are not supported.
//
// Succeeds iff the regular expression matched AND argument parsing was
// successful. Otherwise, returns an err with details of the failure,
// including the original input and regular expression.
//
// Panics if the number of arguments is not consistent with the regular
// expression.
func MatchString(r *regexp.Regexp, s string, args ...interface{}) (err error) {
	if r.NumSubexp() != len(args) {
		panic(fmt.Sprintf("Expected %d arguments, got %d", r.NumSubexp(), len(args)))
	}
	submatches := r.FindStringSubmatch(s)
	if submatches == nil {
		return fmt.Errorf(
			"regular expression did not match.\n\n"+
				"regex: %#v\n"+
				"input: %#v", r.String(), s)
	}
	for i, arg := range args {
		err := save(submatches[i+1], arg)
		if err != nil {
			var buffer bytes.Buffer
			buffer.WriteString(fmt.Sprintf(
				"submatch %d save failed.\n\n"+
					"regex: %#v\n"+
					"input: %#v\n",
				i+1, r.String(), s))
			for i, submatch := range submatches {
				buffer.WriteString(fmt.Sprintf("\nsubmatch %d: %#v", i, submatch))
			}
			return errors.New(buffer.String())
		}
	}
	return nil
}
