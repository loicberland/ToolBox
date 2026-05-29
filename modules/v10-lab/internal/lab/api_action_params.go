package lab

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func intParam(params map[string]any, key string) int {
	value, ok := numberParam(params, key).(int64)
	if ok {
		return int(value)
	}
	switch typed := numberParam(params, key).(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case json.Number:
		integer, _ := typed.Int64()
		return int(integer)
	}
	return 0
}

func numberListParam(params map[string]any, key string) []int {
	items, _ := numberListParamStrict(params, key)
	return items
}

func numberListParamStrict(params map[string]any, key string) ([]int, bool) {
	raw, exists := params[key]
	if !exists || raw == nil {
		return []int{}, true
	}
	switch typed := raw.(type) {
	case []int:
		return append([]int{}, typed...), true
	case []int64:
		items := make([]int, 0, len(typed))
		for _, item := range typed {
			items = append(items, int(item))
		}
		return items, true
	case []float64:
		items := make([]int, 0, len(typed))
		for _, item := range typed {
			if math.IsNaN(item) || math.IsInf(item, 0) || item != math.Trunc(item) {
				return []int{}, false
			}
			items = append(items, int(item))
		}
		return items, true
	case []any:
		items := []int{}
		for _, item := range typed {
			if value, ok := anyToInt(item); ok {
				items = append(items, value)
			} else {
				return []int{}, false
			}
		}
		return items, true
	case string:
		return numberListFromStringStrict(typed)
	default:
		if value, ok := anyToInt(raw); ok {
			return []int{value}, true
		}
		return []int{}, false
	}
}

func objectArrayParam(params map[string]any, key string) []map[string]any {
	raw, exists := params[key]
	if !exists || raw == nil {
		return []map[string]any{}
	}
	switch typed := raw.(type) {
	case []map[string]any:
		return typed
	case []any:
		items := []map[string]any{}
		for _, item := range typed {
			if row, ok := item.(map[string]any); ok {
				items = append(items, row)
			}
		}
		return items
	case string:
		if strings.TrimSpace(typed) == "" {
			return []map[string]any{}
		}
		var rows []map[string]any
		if err := json.Unmarshal([]byte(typed), &rows); err == nil {
			return rows
		}
	}
	return []map[string]any{}
}

func numberListJSONParam(params map[string]any, key string) string {
	payload, err := json.Marshal(numberListParam(params, key))
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func validateNumberListMin(params map[string]any, key string, min int) error {
	items, ok := numberListParamStrict(params, key)
	if !ok {
		return fmt.Errorf("%s: liste de nombres invalide", key)
	}
	for _, item := range items {
		if item < min {
			return fmt.Errorf("Les IDs groupes machine doivent être supérieurs à 0.")
		}
	}
	return nil
}

func anyToInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || typed != math.Trunc(typed) {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		integer, err := typed.Int64()
		if err == nil {
			return int(integer), true
		}
		decimal, err := typed.Float64()
		if err == nil && decimal == math.Trunc(decimal) {
			return int(decimal), true
		}
	case string:
		integer, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return integer, true
		}
	}
	return 0, false
}

func numberListFromString(value string) []int {
	items, _ := numberListFromStringStrict(value)
	return items
}

func numberListFromStringStrict(value string) ([]int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return []int{}, true
	}
	if strings.HasPrefix(value, "[") {
		var raw []any
		if err := json.Unmarshal([]byte(value), &raw); err == nil {
			items := []int{}
			for _, item := range raw {
				if integer, ok := anyToInt(item); ok {
					items = append(items, integer)
				} else {
					return []int{}, false
				}
			}
			return items, true
		}
		return []int{}, false
	}
	items := []int{}
	for _, part := range strings.Split(value, ",") {
		integer, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return []int{}, false
		}
		items = append(items, integer)
	}
	return items, true
}

func queryNumberParam(params map[string]any, key string) string {
	return fmt.Sprint(numberParam(params, key))
}
