package conv

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func Parse[T any](w http.ResponseWriter, r *http.Request) *T {
	var obj T
	if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return nil
	}
	return &obj
}

type PathType interface {
	string | int | uint64
}

func ParseValue[T PathType](r *http.Request, key string) T {
	val := r.PathValue(key)
	var obj T
	switch any(obj).(type) {
	case int:
		iv, err := strconv.Atoi(val)
		if err != nil {
			return obj
		}
		return any(iv).(T)
	case uint64:
		iv, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return obj
		}
		return any(iv).(T)
	case string:
		return any(val).(T)
	}
	return obj
}

func Index[E comparable](s []E, v E) int {
	for i, vs := range s {
		if v == vs {
			return i
		}
	}
	return -1
}

func ToString(el interface{}) string {
	if el == nil {
		return ""
	}
	switch el.(type) {
	case string:
		return el.(string)
	case bool:
		if el.(bool) {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(el.(int))
	case int64:
		return strconv.FormatInt(el.(int64), 10)
	case uint64:
		return strconv.FormatUint(el.(uint64), 10)
	case float32:
		return strconv.FormatFloat(float64(el.(float32)), 'f', -1, 10)
	case float64:
		return strconv.FormatFloat(el.(float64), 'f', -1, 10)
	default:
		return ""
	}
}
func First[T any](t T, _ interface{}) T {
	return t
}
func ToInt(el interface{}) int {
	if el == nil {
		return 0
	}
	switch el.(type) {
	case string:
		return First(strconv.Atoi(el.(string)))
	case bool:
		if el.(bool) {
			return 1
		}
		return 0
	case int:
		return el.(int)
	case int64:
		return int(el.(int64))
	case uint64:
		return int(el.(uint64))
	case float32:
		return int(el.(float32))
	case float64:
		return int(el.(float64))
	default:
		return 0
	}
}

func ToUint(el interface{}) uint64 {
	if el == nil {
		return 0
	}
	switch el.(type) {
	case string:
		return First(strconv.ParseUint(el.(string), 10, 64))
	case bool:
		if el.(bool) {
			return 1
		}
		return 0
	case int:
		return uint64(el.(int))
	case int64:
		return uint64(el.(int64))
	case uint64:
		return el.(uint64)
	case float32:
		return uint64(el.(float32))
	case float64:
		return uint64(el.(float64))
	default:
		return 0
	}
}

func ToBool(el interface{}) bool {
	if el == nil {
		return false
	}
	switch el.(type) {
	case string:
		return strings.ToLower(el.(string)) == "true"
	case bool:
		return el.(bool)
	case int:
		return el.(int) > 0
	case int64:
		return int(el.(int64)) > 0
	case uint64:
		return int(el.(uint64)) > 0
	case float32:
		return int(el.(float32)) > 0
	case float64:
		return int(el.(float64)) > 0
	default:
		return false
	}
}
func FirstQuery[T any](lst []string) *T {
	if lst == nil {
		return nil
	}
	var result T
	value := lst[0]
	switch any(result).(type) {
	case string:
		return any(&value).(*T)
	case int:
		val := ToInt(value)
		return any(&val).(*T)
	case uint64:
		val := ToUint(value)
		return any(&val).(*T)
	case time.Time:
		t1, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil
		}
		return any(&t1).(*T)
	}
	return nil

}
func FirstEl[T any](lst []any) *T {
	if lst == nil {
		return nil
	}
	var result T
	value := lst[0]
	switch v := value.(type) {
	case T:
		return &v
	case string:
		switch any(result).(type) {
		case time.Time:
			t1, err := time.Parse(time.RFC3339, "2023-12-25T15:04:05Z")
			if err != nil {
				return nil
			}
			return any(&t1).(*T)
		}
	}
	return nil
}
