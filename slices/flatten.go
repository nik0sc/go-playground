package slices

import "reflect"

// Flatten turns a two-dimensional slice into a one-dimensional one.
// Example:
// { {1, 2, 3}, {4, 5}, {}, {6} } -> {1, 2, 3, 4, 5, 6}
func Flatten[E any, S ~[]E, SS ~[]S](super SS, flat S) []E {
	if flat == nil {
		flat = make([]E, 0, len(super))
	}

	for _, sl := range super {
		flat = append(flat, sl...)
	}

	return flat
}

// FlattenDeep turns an n-dimensional slice into a one-dimensional one.
// The type parameter E is the element type of the flattened slice.
// Any element of slice must be either E or another slice.
// This rule applies recursively.
func FlattenDeep[E any](slice any) []E {
	var out []E
	eltype := reflect.TypeOf(out).Elem()
	rv := reflect.ValueOf(slice)
	flattenReflect(rv, eltype, &out)
	return out
}

// Actually, slice-like objects are supported too.
// They just need to support reflect Len and Index.
func flattenReflect[E any](rv reflect.Value, eltype reflect.Type, appendTo *[]E) {
	for {
		if !rv.IsValid() {
			// could get this by following untyped nil
			return
		}

		if rv.Type() == eltype {
			el := rv.Interface().(E)
			*appendTo = append(*appendTo, el)
			return
		}

		rvkind := rv.Kind()
		if rvkind == reflect.Interface || rvkind == reflect.Pointer {
			rv = rv.Elem()
		} else {
			break
		}
	}

	for i := 0; i < rv.Len(); i++ {
		flattenReflect(rv.Index(i), eltype, appendTo)
	}
}
