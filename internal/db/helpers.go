package db

import (
	"fmt"
	"log"
	"strings"
	"time"
)

func vectorToString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatIntArray(arr []int) string {
	if len(arr) == 0 {
		return "[]"
	}
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func parseIntArray(s string) []int {
	s = strings.Trim(s, "[]")
	if s == "" {
		return []int{}
	}
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		var v int
		if _, err := fmt.Sscanf(strings.TrimSpace(p), "%d", &v); err == nil {
			result = append(result, v)
		}
	}
	return result
}

// parsePageNumbers handles DuckDB's various return types for INT[] columns
func parsePageNumbers(iface interface{}) []int {
	switch v := iface.(type) {
	case nil:
		return []int{}
	case []interface{}:
		result := make([]int, 0, len(v))
		for _, item := range v {
			switch n := item.(type) {
			case int:
				result = append(result, n)
			case int64:
				result = append(result, int(n))
			case int32:
				result = append(result, int(n))
			case float64:
				result = append(result, int(n))
			default:
				log.Printf("⚠️  parsePageNumbers: unexpected type %T", n)
			}
		}
		return result
	case string:
		return parseIntArray(v)
	default:
		log.Printf("⚠️  parsePageNumbers: unhandled type %T", iface)
		return []int{}
	}
}

func parseTime(s string) (time.Time, error) {
	for _, fmt := range []string{"2006-01-02 15:04:05", time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(fmt, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse time: %s", s)
}

func Float32ToFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i := range v {
		out[i] = float64(v[i])
	}
	return out
}

func Float64ToFloat32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i := range v {
		out[i] = float32(v[i])
	}
	return out
}

func Clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func ClampF(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
