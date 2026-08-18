// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package events

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// formatWithoutPointerAddresses returns a human-readable string of v where:
// - Pointers are dereferenced to print their underlying values (no memory addresses)
// - Cycles are detected and annotated to avoid infinite recursion
// - Output is bounded by a reasonable max depth to prevent excessive logs
func formatWithoutPointerAddresses(v any) string {
	const maxDepth = 5
	var b strings.Builder
	seen := map[uintptr]bool{}
	writeValue(&b, reflect.ValueOf(v), seen, 0, maxDepth)
	return b.String()
}

func writeValue(b *strings.Builder, rv reflect.Value, seen map[uintptr]bool, depth, maxDepth int) {
	if !rv.IsValid() {
		b.WriteString("nil")
		return
	}
	if depth > maxDepth {
		b.WriteString("<max-depth>")
		return
	}

	// Check if rv implements database/sql/driver.Valuer interface before unwrapping
	// But skip nil pointers to avoid panics when calling Value() on nil receivers
	if rv.Kind() != reflect.Pointer || !rv.IsNil() {
		if rv.CanInterface() {
			if v, ok := rv.Interface().(driver.Valuer); ok {
				val, err := v.Value()
				if err == nil {
					writeValue(b, reflect.ValueOf(val), seen, depth+1, maxDepth)
					return
				}
			}
		}
	}

	// Unwrap interfaces
	if rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			b.WriteString("nil")
			return
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Pointer:
		if rv.IsNil() {
			b.WriteString("nil")
			return
		}
		// Cycle detection for pointers
		ptr := rv.Pointer()
		if ptr != 0 {
			if seen[ptr] {
				b.WriteString("<cycle>")
				return
			}
			seen[ptr] = true
			defer delete(seen, ptr)
		}
		writeValue(b, rv.Elem(), seen, depth+1, maxDepth)
		return

	case reflect.Struct:
		b.WriteString(rv.Type().String())
		b.WriteString("{")
		n := rv.NumField()
		first := true
		for i := 0; i < n; i++ {
			tf := rv.Type().Field(i)
			// Skip unexported fields we can't safely interface
			if tf.PkgPath != "" { // unexported
				continue
			}
			if !first {
				b.WriteString(", ")
			}
			first = false
			b.WriteString(tf.Name)
			b.WriteString(": ")
			writeValue(b, rv.Field(i), seen, depth+1, maxDepth)
		}
		b.WriteString("}")
		return

	case reflect.Slice, reflect.Array:
		b.WriteString(rv.Type().String())
		b.WriteString("{")
		l := rv.Len()
		for i := 0; i < l; i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			writeValue(b, rv.Index(i), seen, depth+1, maxDepth)
		}
		b.WriteString("}")
		return

	case reflect.Map:
		b.WriteString(rv.Type().String())
		b.WriteString("{")
		keys := rv.MapKeys()
		// Try to sort keys for determinism where possible
		sort.SliceStable(keys, func(i, j int) bool {
			return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])
		})
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			writeValue(b, k, seen, depth+1, maxDepth)
			b.WriteString(": ")
			writeValue(b, rv.MapIndex(k), seen, depth+1, maxDepth)
		}
		b.WriteString("}")
		return

	case reflect.String:
		b.WriteString(strconv.Quote(rv.String()))
		return

	case reflect.Bool:
		if rv.Bool() {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
		return

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b.WriteString(strconv.FormatInt(rv.Int(), 10))
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		b.WriteString(strconv.FormatUint(rv.Uint(), 10))
		return
	case reflect.Float32, reflect.Float64:
		b.WriteString(strconv.FormatFloat(rv.Float(), 'f', -1, rv.Type().Bits()))
		return
	case reflect.Complex64, reflect.Complex128:
		c := rv.Complex()
		b.WriteString("(")
		b.WriteString(strconv.FormatFloat(real(c), 'f', -1, rv.Type().Bits()/2))
		b.WriteString("+")
		b.WriteString(strconv.FormatFloat(imag(c), 'f', -1, rv.Type().Bits()/2))
		b.WriteString("i)")
		return

	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		// Avoid printing addresses for these kinds; just print the type
		b.WriteString("<")
		b.WriteString(rv.Type().String())
		b.WriteString(">")
		return
	}

	// Fallback: try to use Stringer if available
	if rv.CanInterface() {
		if s, ok := rv.Interface().(fmt.Stringer); ok {
			b.WriteString(s.String())
			return
		}
		b.WriteString(fmt.Sprint(rv.Interface()))
		return
	}
	// Last resort: type name
	b.WriteString(rv.Type().String())
}
