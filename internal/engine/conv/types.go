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
		_, _ = w.Write([]byte(err.Error()))
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
		if v, ok := any(iv).(T); ok {
			return v
		}
		return obj
	case uint64:
		iv, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return obj
		}
		if v, ok := any(iv).(T); ok {
			return v
		}
		return obj
	case string:
		if v, ok := any(val).(T); ok {
			return v
		}
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
	switch v := el.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
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
	switch v := el.(type) {
	case string:
		return First(strconv.Atoi(v))
	case bool:
		if v {
			return 1
		}
		return 0
	case int:
		return v
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func ToUint(el interface{}) uint64 {
	if el == nil {
		return 0
	}
	switch v := el.(type) {
	case string:
		return First(strconv.ParseUint(v, 10, 64))
	case bool:
		if v {
			return 1
		}
		return 0
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	case uint64:
		return v
	case float32:
		return uint64(v)
	case float64:
		return uint64(v)
	default:
		return 0
	}
}

func ToBool(el interface{}) bool {
	if el == nil {
		return false
	}
	switch v := el.(type) {
	case string:
		return strings.EqualFold(v, "true")
	case bool:
		return v
	case int:
		return v > 0
	case int64:
		return int(v) > 0
	case uint64:
		return int(v) > 0
	case float32:
		return int(v) > 0
	case float64:
		return int(v) > 0
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
	case string:
		if _, ok := any(result).(time.Time); ok {
			t1, err := time.Parse(time.RFC3339, "2023-12-25T15:04:05Z")
			if err != nil {
				return nil
			}
			return any(&t1).(*T)
		}
	case T:
		return &v
	}
	return nil
}

func GetOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func ParseInt(s, def string) int64 {
	if s == "" {
		s = def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		v, err = strconv.ParseInt(def, 10, 64)
		if err != nil {
			return 0
		}
	}
	return v
}
