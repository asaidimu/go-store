package utils

import (
	"fmt"
	"reflect"
)

// CopyDocument creates a deep copy of a document.
func CopyDocument(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = copyValue(v)
	}
	return dst
}

// copyValue creates a deep copy of a value, handling nested structures.
func copyValue(src any) any {
	switch v := src.(type) {
	case map[string]any:
		return CopyDocument(v)
	case []any:
		dst := make([]any, len(v))
		for i, elem := range v {
			dst[i] = copyValue(elem)
		}
		return dst
	case []int:
		dst := make([]int, len(v))
		copy(dst, v)
		return dst
	case []string:
		dst := make([]string, len(v))
		copy(dst, v)
		return dst
	default:
		// For primitive types, direct assignment is sufficient
		return v
	}
}

// CompareValues compares two values for B-tree ordering.
func CompareValues(a, b any) int {
	// Handle nil values
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Handle numeric types
	if aIsNum, bIsNum := isNumber(a), isNumber(b); aIsNum && bIsNum {
		return compareNumbers(a, b)
	}

	// Handle same types
	if reflect.TypeOf(a) == reflect.TypeOf(b) {
		return compareSameType(a, b)
	}

	// Handle different types by comparing type names
	typeA, typeB := reflect.TypeOf(a).String(), reflect.TypeOf(b).String()
	if typeA < typeB {
		return -1
	} else if typeA > typeB {
		return 1
	}

	return 0
}

// compareNumbers compares two numeric values.
func compareNumbers(a, b any) int {
	valA := toFloat64(a)
	valB := toFloat64(b)

	if valA < valB {
		return -1
	} else if valA > valB {
		return 1
	}
	return 0
}

// compareSameType compares two values of the same type.
func compareSameType(a, b any) int {
	switch va := a.(type) {
	case string:
		vb := b.(string)
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0

	case bool:
		vb := b.(bool)
		if va == vb {
			return 0
		}
		if va {
			return 1
		}
		return -1

	default:
		// Fallback to string comparison for other types
		strA := fmt.Sprintf("%v", a)
		strB := fmt.Sprintf("%v", b)
		if strA < strB {
			return -1
		} else if strA > strB {
			return 1
		}
		return 0
	}
}

// toFloat64 converts a numeric value to float64.
func toFloat64(v any) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	default:
		return 0 // Should not happen if isNumber returned true
	}
}

// isNumber checks if a value is a numeric type.
func isNumber(v any) bool {
	switch v.(type) {
	case int, int32, int64, float32, float64:
		return true
	default:
		return false
	}
}
