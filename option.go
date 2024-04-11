package mo

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

var optionNoSuchElement = fmt.Errorf("no such element")

// Some builds an Option when value is present.
// Play: https://go.dev/play/p/iqz2n9n0tDM
func Some[T any](value T) Option[T] {
	return Option[T]{
		isPresent: true,
		value:     value,
	}
}

// None builds an Option when value is absent.
// Play: https://go.dev/play/p/yYQPsYCSYlD
func None[T any]() Option[T] {
	return Option[T]{
		isPresent: false,
	}
}

// TupleToOption builds a Some Option when second argument is true, or None.
// Play: https://go.dev/play/p/gkrg2pZwOty
func TupleToOption[T any](value T, ok bool) Option[T] {
	if ok {
		return Some(value)
	}
	return None[T]()
}

// EmptyableToOption builds a Some Option when value is not empty, or None.
// Play: https://go.dev/play/p/GSpQQ-q-UES
func EmptyableToOption[T any](value T) Option[T] {
	// 🤮
	isZero := reflect.ValueOf(&value).Elem().IsZero()
	if isZero {
		return None[T]()
	}

	return Some(value)
}

// PointerToOption builds a Some Option when value is not nil, or None.
// Play: https://go.dev/play/p/yPVMj4DUb-I
func PointerToOption[T any](value *T) Option[T] {
	if value == nil {
		return None[T]()
	}

	return Some(*value)
}

// Option is a container for an optional value of type T. If value exists, Option is
// of type Some. If the value is absent, Option is of type None.
type Option[T any] struct {
	isPresent bool
	value     T
}

// IsPresent returns false when value is absent.
// Play: https://go.dev/play/p/nDqIaiihyCA
func (o Option[T]) IsPresent() bool {
	return o.isPresent
}

// IsAbsent returns false when value is present.
// Play: https://go.dev/play/p/23e2zqyVOQm
func (o Option[T]) IsAbsent() bool {
	return !o.isPresent
}

// Size returns 1 when value is present or 0 instead.
// Play: https://go.dev/play/p/7ixCNG1E9l7
func (o Option[T]) Size() int {
	if o.isPresent {
		return 1
	}

	return 0
}

// Get returns value and presence.
// Play: https://go.dev/play/p/0-JBa1usZRT
func (o Option[T]) Get() (T, bool) {
	if !o.isPresent {
		return empty[T](), false
	}

	return o.value, true
}

// MustGet returns value if present or panics instead.
// Play: https://go.dev/play/p/RVBckjdi5WR
func (o Option[T]) MustGet() T {
	if !o.isPresent {
		panic(optionNoSuchElement)
	}

	return o.value
}

// OrElse returns value if present or default value.
// Play: https://go.dev/play/p/TrGByFWCzXS
func (o Option[T]) OrElse(fallback T) T {
	if !o.isPresent {
		return fallback
	}

	return o.value
}

// OrEmpty returns value if present or empty value.
// Play: https://go.dev/play/p/SpSUJcE-tQm
func (o Option[T]) OrEmpty() T {
	return o.value
}

// ForEach executes the given side-effecting function of value is present.
func (o Option[T]) ForEach(onValue func(value T)) {
	if o.isPresent {
		onValue(o.value)
	}
}

// Match executes the first function if value is present and second function if absent.
// It returns a new Option.
// Play: https://go.dev/play/p/1V6st3LDJsM
func (o Option[T]) Match(onValue func(value T) (T, bool), onNone func() (T, bool)) Option[T] {
	if o.isPresent {
		return TupleToOption(onValue(o.value))
	}
	return TupleToOption(onNone())
}

// Map executes the mapper function if value is present or returns None if absent.
// Play: https://go.dev/play/p/mvfP3pcP_eJ
func (o Option[T]) Map(mapper func(value T) T) Option[T] {
	if o.isPresent {
		return TupleToOption(mapper(o.value), true)
	}

	return None[T]()
}

// MapNone executes the mapper function if value is absent or returns Option.
// Play: https://go.dev/play/p/_KaHWZ6Q17b
func (o Option[T]) MapNone(mapper func() (T, bool)) Option[T] {
	if o.isPresent {
		return Some(o.value)
	}

	return TupleToOption(mapper())
}

// FlatMap executes the mapper function if value is present or returns None if absent.
// Play: https://go.dev/play/p/OXO-zJx6n5r
func (o Option[T]) FlatMap(mapper func(value T) Option[T]) Option[T] {
	if o.isPresent {
		return mapper(o.value)
	}

	return None[T]()
}

// ToPointer returns value if present or a nil pointer.
// Play: https://go.dev/play/p/N43w92SM-Bs
func (o Option[T]) ToPointer() *T {
	if !o.isPresent {
		return nil
	}

	return &o.value
}

// MarshalJSON encodes Option into json.
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if o.isPresent {
		return json.Marshal(o.value)
	}

	// if anybody find a way to support `omitempty` param, please contribute!
	return json.Marshal(nil)
}

// UnmarshalJSON decodes Option from json.
func (o *Option[T]) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		o.isPresent = false
		return nil
	}

	err := json.Unmarshal(b, &o.value)
	if err != nil {
		return err
	}

	o.isPresent = true
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (o Option[T]) MarshalText() ([]byte, error) {
	return json.Marshal(o)
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (o *Option[T]) UnmarshalText(data []byte) error {
	return json.Unmarshal(data, o)
}

// MarshalBinary is the interface implemented by an object that can marshal itself into a binary form.
func (o Option[T]) MarshalBinary() ([]byte, error) {
	if !o.isPresent {
		return []byte{0}, nil
	}

	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.value); err != nil {
		return []byte{}, err
	}

	return append([]byte{1}, buf.Bytes()...), nil
}

// UnmarshalBinary is the interface implemented by an object that can unmarshal a binary representation of itself.
func (o *Option[T]) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return errors.New("Option[T].UnmarshalBinary: no data")
	}

	if data[0] == 0 {
		o.isPresent = false
		o.value = empty[T]()
		return nil
	}

	buf := bytes.NewBuffer(data[1:])
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&o.value)
	if err != nil {
		return err
	}

	o.isPresent = true
	return nil
}

// GobEncode implements the gob.GobEncoder interface.
func (o Option[T]) GobEncode() ([]byte, error) {
	return o.MarshalBinary()
}

// GobDecode implements the gob.GobDecoder interface.
func (o *Option[T]) GobDecode(data []byte) error {
	return o.UnmarshalBinary(data)
}

// Scan implements the SQL sql.Scanner interface.
func (o *Option[T]) Scan(src any) error {
	if src == nil {
		o.isPresent = false
		o.value = empty[T]()
		return nil
	}

	// is is only possible to assert interfaces, so convert first
	// https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#why-not-permit-type-assertions-on-values-whose-type-is-a-type-parameter
	var t T
	if tScanner, ok := interface{}(&t).(sql.Scanner); ok {
		if err := tScanner.Scan(src); err != nil {
			return fmt.Errorf("failed to scan: %w", err)
		}

		o.isPresent = true
		o.value = t
		return nil
	}

	if err := convertAssign(&t, src); err == nil {
		o.isPresent = true
		o.value = t
		return nil
	}

	return fmt.Errorf("failed to scan Option[T]")
}

// Value implements the driver Valuer interface.
func (o Option[T]) Value() (driver.Value, error) {
	if !o.isPresent {
		return nil, nil
	}

	return driver.DefaultParameterConverter.ConvertValue(o.value), nil
}

// //////////
// Shamelessly extracted from https://github.com/golang/go/blob/master/src/database/sql/convert.go
// //////////
var errNilPtr = errors.New("destination pointer is nil") // embedded in descriptive error

type decimalDecompose interface {
	// Decompose returns the internal decimal state in parts.
	// If the provided buf has sufficient capacity, buf may be returned as the coefficient with
	// the value set and length set as appropriate.
	Decompose(buf []byte) (form byte, negative bool, coefficient []byte, exponent int32)
}

type decimalCompose interface {
	// Compose sets the internal decimal value from parts. If the value cannot be
	// represented then an error should be returned.
	Compose(form byte, negative bool, coefficient []byte, exponent int32) error
}

func convertAssign(dest, src any) error {
	// Common cases, without reflect.
	switch s := src.(type) {
	case string:
		switch d := dest.(type) {
		case *string:
			if d == nil {
				return errNilPtr
			}
			*d = s
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = []byte(s)
			return nil
		}
	case []byte:
		switch d := dest.(type) {
		case *string:
			if d == nil {
				return errNilPtr
			}
			*d = string(s)
			return nil
		case *any:
			if d == nil {
				return errNilPtr
			}
			*d = bytes.Clone(s)
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = bytes.Clone(s)
			return nil
		}
	case time.Time:
		switch d := dest.(type) {
		case *time.Time:
			*d = s
			return nil
		case *string:
			*d = s.Format(time.RFC3339Nano)
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = []byte(s.Format(time.RFC3339Nano))
			return nil
		}
	case decimalDecompose:
		switch d := dest.(type) {
		case decimalCompose:
			return d.Compose(s.Decompose(nil))
		}
	case nil:
		switch d := dest.(type) {
		case *any:
			if d == nil {
				return errNilPtr
			}
			*d = nil
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = nil
			return nil
		}
	}

	var sv reflect.Value

	switch d := dest.(type) {
	case *string:
		sv = reflect.ValueOf(src)
		switch sv.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			*d = asString(src)
			return nil
		}
	case *[]byte:
		sv = reflect.ValueOf(src)
		if b, ok := asBytes(nil, sv); ok {
			*d = b
			return nil
		}
	case *bool:
		bv, err := driver.Bool.ConvertValue(src)
		if err == nil {
			*d = bv.(bool)
		}
		return err
	case *any:
		*d = src
		return nil
	}

	if scanner, ok := dest.(sql.Scanner); ok {
		return scanner.Scan(src)
	}

	dpv := reflect.ValueOf(dest)
	if dpv.Kind() != reflect.Pointer {
		return errors.New("destination not a pointer")
	}
	if dpv.IsNil() {
		return errNilPtr
	}

	if !sv.IsValid() {
		sv = reflect.ValueOf(src)
	}

	dv := reflect.Indirect(dpv)
	if sv.IsValid() && sv.Type().AssignableTo(dv.Type()) {
		switch b := src.(type) {
		case []byte:
			dv.Set(reflect.ValueOf(bytes.Clone(b)))
		default:
			dv.Set(sv)
		}
		return nil
	}

	if dv.Kind() == sv.Kind() && sv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(sv.Convert(dv.Type()))
		return nil
	}

	// The following conversions use a string value as an intermediate representation
	// to convert between various numeric types.
	//
	// This also allows scanning into user defined types such as "type Int int64".
	// For symmetry, also check for string destination types.
	switch dv.Kind() {
	case reflect.Pointer:
		if src == nil {
			dv.SetZero()
			return nil
		}
		dv.Set(reflect.New(dv.Type().Elem()))
		return convertAssign(dv.Interface(), src)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if src == nil {
			return fmt.Errorf("converting NULL to %s is unsupported", dv.Kind())
		}
		s := asString(src)
		i64, err := strconv.ParseInt(s, 10, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if src == nil {
			return fmt.Errorf("converting NULL to %s is unsupported", dv.Kind())
		}
		s := asString(src)
		u64, err := strconv.ParseUint(s, 10, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetUint(u64)
		return nil
	case reflect.Float32, reflect.Float64:
		if src == nil {
			return fmt.Errorf("converting NULL to %s is unsupported", dv.Kind())
		}
		s := asString(src)
		f64, err := strconv.ParseFloat(s, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetFloat(f64)
		return nil
	case reflect.String:
		if src == nil {
			return fmt.Errorf("converting NULL to %s is unsupported", dv.Kind())
		}
		switch v := src.(type) {
		case string:
			dv.SetString(v)
			return nil
		case []byte:
			dv.SetString(string(v))
			return nil
		}
	}

	return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type %T", src, dest)
}

func strconvErr(err error) error {
	if ne, ok := err.(*strconv.NumError); ok {
		return ne.Err
	}
	return err
}

func asString(src any) string {
	switch v := src.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	rv := reflect.ValueOf(src)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 32)
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	}
	return fmt.Sprintf("%v", src)
}

func asBytes(buf []byte, rv reflect.Value) (b []byte, ok bool) {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(buf, rv.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.AppendUint(buf, rv.Uint(), 10), true
	case reflect.Float32:
		return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 32), true
	case reflect.Float64:
		return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 64), true
	case reflect.Bool:
		return strconv.AppendBool(buf, rv.Bool()), true
	case reflect.String:
		s := rv.String()
		return append(buf, s...), true
	}
	return
}
