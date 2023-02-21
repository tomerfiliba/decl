package env

import (
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func LoadEnvSpec(envSpec interface{}) {
	LoadEnvSpecFrom(envSpec, os.Getenv)
}

func LoadEnvSpecFrom(envSpec interface{}, getenv func(string) string) {
	tp := reflect.TypeOf(envSpec).Elem()
	val := reflect.ValueOf(envSpec).Elem()

	for i := 0; i < tp.NumField(); i++ {
		f := tp.Field(i)
		tag := f.Tag.Get("env")
		if tag == "" {
			continue
		}
		parts := strings.SplitN(tag, "=", 2)
		envName := strings.TrimSpace(parts[0])
		if envName == "" {
			log.Panicf("Missing env var name '%s'", tag)
		}

		fv := val.Field(i)

		if envName == "*" {
			// nested structs
			LoadEnvSpecFrom(fv.Interface(), getenv)
			continue
		}

		envVal := getenv(envName)
		if envVal == "" {
			if len(parts) == 1 {
				log.Panicf("missing required env var %s", envName)
			} else {
				// no value, keep the default
				envVal = parts[1]
			}
		}

		switch fv.Interface().(type) {
		case string:
			fv.SetString(envVal)

		case bool:
			envVal = strings.ToLower(strings.TrimSpace(envVal))
			switch envVal {
			case "t", "y", "true", "yes", "1":
				fv.SetBool(true)
			case "f", "n", "false", "no", "0":
				fv.SetBool(false)
			default:
				log.Panicf("parsing %s: %s", envName, envVal)
			}

		case int, int8, int16, int32, int64:
			v, err := strconv.ParseInt(strings.TrimSpace(envVal), 10, 64)
			if err != nil {
				log.Panicf("parsing %s: %v", envName, err)
			}
			fv.SetInt(v)

		case uint, uint8, uint16, uint32, uint64:
			v, err := strconv.ParseUint(strings.TrimSpace(envVal), 10, 64)
			if err != nil {
				log.Panicf("parsing %s: %v", envName, err)
			}
			fv.SetUint(v)

		case float32, float64:
			v, err := strconv.ParseFloat(strings.TrimSpace(envVal), 64)
			if err != nil {
				log.Panicf("parsing %s: %v", envName, err)
			}
			fv.SetFloat(v)

		case time.Duration:
			dur, err := time.ParseDuration(strings.TrimSpace(envVal))
			if err != nil {
				v, err2 := strconv.ParseInt(strings.TrimSpace(envVal), 10, 64)
				if err2 != nil {
					log.Panicf("parsing %s: %v", envName, err)
				}
				// assume seconds
				dur = time.Duration(v) * time.Second
			}

			fv.SetInt(int64(dur))

		default:
			log.Panicf("Unsupported type %s for %s", f.Type.Name(), f.Name)
		}
	}
}
