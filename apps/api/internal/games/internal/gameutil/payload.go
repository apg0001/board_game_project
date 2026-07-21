package gameutil

import (
	"encoding/json"
	"math"
)

func Int(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		if typed > uint64(maxInt()) {
			return 0, false
		}
		return int(typed), true
	case float32:
		value := float64(typed)
		if math.Trunc(value) != value {
			return 0, false
		}
		return int(value), true
	case float64:
		if math.Trunc(typed) != typed {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		if integer, err := typed.Int64(); err == nil {
			return int(integer), true
		}
		if decimal, err := typed.Float64(); err == nil && math.Trunc(decimal) == decimal {
			return int(decimal), true
		}
	}
	return 0, false
}

func maxInt() int {
	return int(^uint(0) >> 1)
}
