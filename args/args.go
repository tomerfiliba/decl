package args

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func LoadArgsSpec(argsSpec interface{}) error {
	return LoadArgsSpecFrom(argsSpec, os.Args)
}

func LoadArgsSpecFrom(argsSpec interface{}, osArgs []string) error {
	tp := reflect.TypeOf(argsSpec).Elem()
	val := reflect.ValueOf(argsSpec).Elem()

	missing := make(map[int]bool)
	type ArgInfo struct {
		fieldIdx int
		needsVal bool
	}
	longSwitches := make(map[string]ArgInfo)
	shortSwitches := make(map[string]ArgInfo)
	wasSet := make(map[int]bool)
	var positional []int
	captureAllPositional := -1

	var setSwitch = func(fieldIdx int, swVal string) error {
		delete(missing, fieldIdx)
		f := tp.Field(fieldIdx)
		fv := val.Field(fieldIdx)

		isSlice := false
		switch fv.Type().Kind() {
		case reflect.Slice:
			isSlice = true
		}

		if isSlice && wasSet[fieldIdx] {
			return fmt.Errorf("Field %s was already set", f.Name)
		}
		wasSet[fieldIdx] = true

		if isSlice {
			return fmt.Errorf("Arrays are unsupported (%s %s)", f.Name, f.Type.String())
		}

		switch fv.Interface().(type) {
		case bool:
			swVal = strings.ToLower(strings.TrimSpace(swVal))
			switch swVal {
			case "":
				fv.SetBool(!fv.Bool())
			case "true", "t", "yes", "y", "1":
				fv.SetBool(true)
			case "false", "f", "no", "n", "0":
				fv.SetBool(false)
			default:
				return fmt.Errorf("Cannot parse %v (%s)", swVal, f.Name)
			}

		case string:
			fv.SetString(swVal)

		case int, int8, int16, int32, int64:
			v, err := strconv.ParseInt(strings.TrimSpace(swVal), 10, 64)
			if err != nil {
				return fmt.Errorf("Cannot parse %v (%s) - %v", swVal, f.Name, err)
			}
			fv.SetInt(v)

		case uint, uint8, uint16, uint32, uint64:
			v, err := strconv.ParseUint(strings.TrimSpace(swVal), 10, 64)
			if err != nil {
				return fmt.Errorf("Cannot parse %v (%s) - %v", swVal, f.Name, err)
			}
			fv.SetUint(v)

		case float32, float64:
			v, err := strconv.ParseFloat(strings.TrimSpace(swVal), 64)
			if err != nil {
				return fmt.Errorf("Cannot parse %v (%s) - %v", swVal, f.Name, err)
			}
			fv.SetFloat(v)

		default:
			return fmt.Errorf("Unsupported type %s (%s): %v", f.Type.String(), f.Name, swVal)
		}

		return nil
	}

	for i := 0; i < tp.NumField(); i++ {
		f := tp.Field(i)
		tag := f.Tag.Get("arg")
		if tag == "" {
			continue
		}
		if tag == "*" {
			if captureAllPositional >= 0 {
				return fmt.Errorf("%s is a capture-all, %s cannot follow", tp.Field(captureAllPositional).Name, f.Name)
			}

			positional = append(positional, i)
			switch f.Type.Kind() {
			case reflect.Slice:
				captureAllPositional = i
			}
			continue
		}

		parts := strings.SplitN(tag, "=", 2)
		if len(parts) == 2 {
			if err := setSwitch(i, parts[1]); err != nil {
				return err
			}
		} else {
			missing[i] = true
		}

		for _, sw := range strings.Split(parts[0], ",") {
			if len(sw) == 1 {
				fv := val.Field(i)
				switch fv.Interface().(type) {
				case bool:
					delete(missing, i)
					shortSwitches[sw] = ArgInfo{i, false}
				default:
					shortSwitches[sw] = ArgInfo{i, true}
				}
			} else {
				fv := val.Field(i)
				switch fv.Interface().(type) {
				case bool:
					delete(missing, i)
					longSwitches[sw] = ArgInfo{i, false}
				default:
					longSwitches[sw] = ArgInfo{i, true}
				}
			}
		}
	}

	var posArgs []string
	for i := 1; i < len(osArgs); i++ {
		a := osArgs[i]
		if a == "--" {
			posArgs = append(posArgs, osArgs[i+1:]...)
			break
		}

		if strings.HasPrefix(a, "--") {
			name := a[2:]
			parts := strings.SplitN(name, "=", 2)
			if len(parts) == 2 {
				name = parts[0]
			}
			info, ok := longSwitches[name]
			if !ok {
				return fmt.Errorf("Unknown switch %s", a)
			}
			var err error
			if len(parts) == 2 {
				if !info.needsVal {
					return fmt.Errorf("Switch %s does not expect a value", a)
				}
				err = setSwitch(info.fieldIdx, parts[1])
			} else if info.needsVal {
				if i == len(osArgs)-1 {
					return fmt.Errorf("Switch %s expects a value", a)
				}
				err = setSwitch(info.fieldIdx, osArgs[i+1])
				i++
			} else {
				err = setSwitch(info.fieldIdx, "")
			}
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(a, "-") {
			if len(a) == 1 {
				return fmt.Errorf("Invalid switch %s", a)
			}
			name := string(a[1])
			info, ok := shortSwitches[name]
			if !ok {
				return fmt.Errorf("Unknown switch %s", a)
			}
			var err error
			if info.needsVal {
				if len(a) > 2 {
					err = setSwitch(info.fieldIdx, a[2:])
				} else {
					if i == len(osArgs)-1 {
						return fmt.Errorf("Switch %s expects a value", a)
					}
					err = setSwitch(info.fieldIdx, osArgs[i+1])
					i++
				}
			} else {
				if len(a) > 2 {
					return fmt.Errorf("Switch %s does not expect a value", a)
				}
				err = setSwitch(info.fieldIdx, "")
			}
			if err != nil {
				return err
			}
		} else {
			posArgs = append(posArgs, a)
			continue
		}
	}

	if len(missing) > 0 {
		names := make([]string, 0, len(missing))
		for i, _ := range missing {
			names = append(names, tp.Field(i).Name)
		}
		return fmt.Errorf("Missing required arguments: %v", names)
	}

	for _, p := range positional {
		if len(posArgs) == 0 {
			return fmt.Errorf("Missing required positional argument %s", tp.Field(p).Name)
		}
		if p == captureAllPositional {
			fv := val.Field(p)
			fv.Set(reflect.ValueOf(posArgs))
			posArgs = nil
		} else {
			a := posArgs[0]
			posArgs = posArgs[1:]
			err := setSwitch(p, a)
			if err != nil {
				return err
			}
		}
	}

	if len(posArgs) > 0 {
		return fmt.Errorf("Unconsumed positional arguments: %v", posArgs)
	}

	return nil
}
