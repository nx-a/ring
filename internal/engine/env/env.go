package env

import (
	"embed"
	"fmt"
	"github.com/nx-a/ring/internal/engine/conv"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Environment struct {
	mutex sync.RWMutex
	data  map[string]interface{}
}

var isEnv = regexp.MustCompile(`\$\{(.*?)\}`)

func New(emb embed.FS) *Environment {
	env := &Environment{
		data: make(map[string]interface{}),
	}
	env.mutex.Lock()
	defer env.mutex.Unlock()
	env.data = convertYamlToProp([]byte(def))
	file, err := emb.ReadFile("config.yml")
	if err != nil {
		return nil
	}
	subenv := convertYamlToProp(file)
	for key, value := range subenv {
		env.data[key] = value
	}
	return env
}

var def = `server:
  host: ${SERVER_HOST:*}
  port: ${SERVER_PORT:':80'}
  maxSize: ${MAX_REQUEST_BODY_SIZE:104857600}
service:
  name: ${SERVICE_NAME:app}
  prod: ${PROD:false}`

type Env struct {
	mutex sync.RWMutex
	data  map[string]interface{}
}

func GetInterface(env *Environment, name string) interface{} {
	subName := strings.Split(name, ".")
	data := env.data
	var val interface{}
	var ok bool
	for i, nameData := range subName {
		val, ok = data[nameData]
		if !ok {
			return nil
		}
		if i < len(subName)-1 {
			typeReflect := reflect.TypeOf(val)
			if typeReflect.Kind() == reflect.Map {
				data = val.(map[string]interface{})
			} else {
				return nil
			}
		}
	}
	return val
}

func (e *Environment) Get(name string) string {
	return GetString(e, name)
}

func GetString(env *Environment, name string) string {
	val := GetInterface(env, name)
	if val == nil {
		return ""
	}
	if reflect.TypeOf(val).Kind() != reflect.String {
		return conv.ToString(val)
	}
	strVal := val.(string)
	return checkEnv(strVal)
}
func checkEnv(value string) string {
	if isEnv.MatchString(value) {
		find := isEnv.FindStringSubmatch(value)
		sub := strings.SplitN(find[1], ":", 2)
		envOs := os.Getenv(sub[0])
		if len(envOs) > 0 {
			return envOs
		}
		if len(sub) > 1 {
			return sub[1]
		}
	}
	return strings.TrimSpace(value)
}

func Convert[T any](env *Environment, name string) T {
	content := GetInterface(env, name)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	var t T
	var val reflect.Value
	typeReflect := reflect.TypeOf(t)
	valueReflect := reflect.TypeOf(content)
	if valueReflect.Kind() == typeReflect.Kind() {
		val = reflect.ValueOf(content)
		reflect.ValueOf(&t).Elem().Set(val)
		return t
	}
	if valueReflect.Kind() != reflect.String {
		log.Println("Convert", name, content, valueReflect.Name(), typeReflect.Name())
		return t
	}
	contentStr := checkEnv(content.(string))

	switch typeReflect.Kind() {
	case reflect.String:
		val = reflect.ValueOf(contentStr)
		reflect.ValueOf(&t).Elem().Set(val)
		return t
	case reflect.Bool:
		if contentStr == "true" {
			val = reflect.ValueOf(true)
		} else {
			val = reflect.ValueOf(false)
		}
	case reflect.Float32:
		f, e := strconv.ParseFloat(strings.Replace(contentStr, ",", ".", -1), 64)
		if e != nil {
			return t
		}
		val = reflect.ValueOf(f)
	case reflect.Float64:
		f, e := strconv.ParseFloat(strings.Replace(contentStr, ",", ".", -1), 64)
		if e != nil {
			return t
		}
		val = reflect.ValueOf(f)
	case reflect.Uint32:
		u, e := strconv.ParseUint(contentStr, 10, 64)
		if e != nil {
			return t
		}
		val = reflect.ValueOf(u)
	case reflect.Uint:
		u, e := strconv.ParseUint(contentStr, 10, 64)
		if e != nil {
			return t
		}
		val = reflect.ValueOf(u)
	case reflect.Int:
		u, e := strconv.Atoi(contentStr)
		if e != nil {
			return t
		}
		val = reflect.ValueOf(u)
	case reflect.Int64:
		u, e := strconv.ParseInt(contentStr, 10, 64)
		if e != nil {
			return t
		}
		val = reflect.ValueOf(u)
	default:
		fmt.Println("Convert.reflect.default:", contentStr)
	}
	reflect.ValueOf(&t).Elem().Set(val)
	return t
}
func Get[T any](env *Environment, name string) T {
	return Convert[T](env, name)
}

func convertYamlToProp(file []byte) map[string]interface{} {
	var local map[string]interface{}
	err := yaml.Unmarshal(file, &local)
	if err != nil {
		return nil
	}
	return local
}
