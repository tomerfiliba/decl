package args

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func LoadArgsSpec(argsSpec interface{}) error {
	return LoadArgsSpecFrom(argsSpec, os.Args)
}

type ArgInfo struct {
	fieldIdx int
	needsVal bool
}

type ParsedArgs struct {
	tp                   reflect.Type
	val                  reflect.Value
	LongSwitches         map[string]ArgInfo
	ShortSwitches        map[string]ArgInfo
	Defaults             map[int]string
	Positional           []int
	Missing              map[int]bool
	CaptureAllPositional int
	HelpMsg              []string
}

func (parsed *ParsedArgs) String() string {
	posNames := make([]string, 0)
	for _, i := range parsed.Positional {
		if i == parsed.CaptureAllPositional {
			posNames = append(posNames, fmt.Sprintf("%s...", parsed.tp.Field(parsed.CaptureAllPositional).Name))
		} else {
			posNames = append(posNames, parsed.tp.Field(i).Name)
		}
	}
	res := fmt.Sprintf("%s [switches] %s\n\n", path.Base(os.Args[0]), strings.Join(posNames, " "))
	for _, line := range parsed.HelpMsg {
		res += line + "\n"
	}
	return res
}

func (parsed *ParsedArgs) ShowHelp() {
	fmt.Println(parsed.String())
	os.Exit(2)
}

func (parsed *ParsedArgs) setSwitch(fieldIdx int, swVal string) error {
	if fieldIdx < 0 {
		// does not return
		parsed.ShowHelp()
		return nil
	}
	wasSet := make(map[int]bool)
	delete(parsed.Missing, fieldIdx)
	f := parsed.tp.Field(fieldIdx)
	fv := parsed.val.Field(fieldIdx)

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

	case time.Duration:
		dur, err := time.ParseDuration(strings.TrimSpace(swVal))
		if err != nil {
			v, err2 := strconv.ParseInt(strings.TrimSpace(swVal), 10, 64)
			if err2 != nil {
				return fmt.Errorf("Cannot parse %v (%s) - %v", swVal, f.Name, err)
			}
			// assume seconds
			dur = time.Duration(v) * time.Second
		}

		fv.SetInt(int64(dur))

	case time.Time:
		for _, layout := range []string{time.RFC3339, time.RFC1123, time.RFC1123Z, time.RFC822, time.RFC822Z,
			time.UnixDate, time.DateTime, time.DateOnly, time.TimeOnly} {
			tm, err := time.Parse(layout, strings.TrimSpace(swVal))
			if err == nil {
				fv.Set(reflect.ValueOf(tm))
				return nil
			}
		}
		return fmt.Errorf("Cannot parse %v (%s) as any known time format", swVal, f.Name)

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

func ParseArgsSpec(argsSpec interface{}) (*ParsedArgs, error) {
	parsed := &ParsedArgs{
		tp:                   reflect.TypeOf(argsSpec).Elem(),
		val:                  reflect.ValueOf(argsSpec).Elem(),
		Missing:              make(map[int]bool),
		LongSwitches:         make(map[string]ArgInfo),
		ShortSwitches:        make(map[string]ArgInfo),
		Defaults:             make(map[int]string),
		Positional:           make([]int, 0),
		HelpMsg:              make([]string, 0),
		CaptureAllPositional: -1,
	}

	for i := 0; i < parsed.tp.NumField(); i++ {
		f := parsed.tp.Field(i)
		tag := f.Tag.Get("arg")
		if tag == "" {
			continue
		}
		if tag == "*" {
			if parsed.CaptureAllPositional >= 0 {
				return nil, fmt.Errorf("%s is a capture-all, %s cannot follow", parsed.tp.Field(parsed.CaptureAllPositional).Name, f.Name)
			}

			parsed.Positional = append(parsed.Positional, i)
			switch f.Type.Kind() {
			case reflect.Slice:
				parsed.CaptureAllPositional = i
			}
			continue
		}

		parts := strings.SplitN(tag, "=", 2)
		if len(parts) == 2 {
			parsed.Defaults[i] = parts[1]
			err := parsed.setSwitch(i, parts[1])
			if err != nil {
				return nil, fmt.Errorf("Setting default value of %v to %v: %v", parsed.tp.Field(i).Name, parts[1], err)
			}
		} else {
			parsed.Missing[i] = true
		}

		shorts := make([]string, 0)
		longs := make([]string, 0)

		for _, sw := range strings.Split(parts[0], ",") {
			if len(sw) == 1 {
				fv := parsed.val.Field(i)
				switch fv.Interface().(type) {
				case bool:
					delete(parsed.Missing, i)
					parsed.ShortSwitches[sw] = ArgInfo{i, false}
				default:
					parsed.ShortSwitches[sw] = ArgInfo{i, true}
				}
				shorts = append(shorts, sw)
			} else {
				fv := parsed.val.Field(i)
				switch fv.Interface().(type) {
				case bool:
					delete(parsed.Missing, i)
					parsed.LongSwitches[sw] = ArgInfo{i, false}
				default:
					parsed.LongSwitches[sw] = ArgInfo{i, true}
				}
				longs = append(longs, sw)
			}
		}
		swParts := make([]string, 0)
		for _, sw := range shorts {
			swParts = append(swParts, "-"+sw)
		}
		for _, sw := range longs {
			swParts = append(swParts, "--"+sw)
		}
		helpLine := fmt.Sprintf("    %-20s ", strings.Join(swParts, " "))
		helpTag := f.Tag.Get("argHelp")
		if helpTag == "" {
			if f.Type.Name() == "bool" {
				helpLine += fmt.Sprintf("Sets '%s'", f.Name)
			} else {
				helpLine += fmt.Sprintf("Sets '%s' (%s)", f.Name, f.Type.Name())
			}
		} else {
			helpLine += helpTag
		}
		if ok := parsed.Missing[i]; ok {
			helpLine += "; required"
		} else if defVal, ok := parsed.Defaults[i]; ok {
			helpLine += fmt.Sprintf("; defaults to %s", defVal)
		}

		parsed.HelpMsg = append(parsed.HelpMsg, helpLine)
	}

	return parsed, nil
}

func LoadArgsSpecFrom(argsSpec interface{}, osArgs []string) error {
	parsed, err := ParseArgsSpec(argsSpec)
	if err != nil {
		return err
	}

	if _, ok := parsed.ShortSwitches["h"]; !ok {
		// add -h as "help switch"
		parsed.ShortSwitches["h"] = ArgInfo{-1, false}
	}
	if _, ok := parsed.LongSwitches["help"]; !ok {
		// add --help as "help switch"
		parsed.ShortSwitches["help"] = ArgInfo{-1, false}
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
			info, ok := parsed.LongSwitches[name]
			if !ok {
				return fmt.Errorf("Unknown switch %s", a)
			}
			var err error
			if len(parts) == 2 {
				if !info.needsVal {
					return fmt.Errorf("Switch %s does not expect a value", a)
				}
				err = parsed.setSwitch(info.fieldIdx, parts[1])
			} else if info.needsVal {
				if i == len(osArgs)-1 {
					return fmt.Errorf("Switch %s expects a value", a)
				}
				err = parsed.setSwitch(info.fieldIdx, osArgs[i+1])
				i++
			} else {
				err = parsed.setSwitch(info.fieldIdx, "")
			}
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(a, "-") {
			if len(a) == 1 {
				return fmt.Errorf("Invalid switch %s", a)
			}
			name := string(a[1])
			info, ok := parsed.ShortSwitches[name]
			if !ok {
				return fmt.Errorf("Unknown switch %s", a)
			}
			var err error
			if info.needsVal {
				if len(a) > 2 {
					err = parsed.setSwitch(info.fieldIdx, a[2:])
				} else {
					if i == len(osArgs)-1 {
						return fmt.Errorf("Switch %s expects a value", a)
					}
					err = parsed.setSwitch(info.fieldIdx, osArgs[i+1])
					i++
				}
			} else {
				if len(a) > 2 {
					return fmt.Errorf("Switch %s does not expect a value", a)
				}
				err = parsed.setSwitch(info.fieldIdx, "")
			}
			if err != nil {
				return err
			}
		} else {
			posArgs = append(posArgs, a)
			continue
		}
	}

	if len(parsed.Missing) > 0 {
		names := make([]string, 0, len(parsed.Missing))
		for i, _ := range parsed.Missing {
			names = append(names, parsed.tp.Field(i).Name)
		}
		return fmt.Errorf("Missing required arguments: %v", names)
	}

	for _, p := range parsed.Positional {
		if len(posArgs) == 0 {
			return fmt.Errorf("Missing required positional argument %s", parsed.tp.Field(p).Name)
		}
		if p == parsed.CaptureAllPositional {
			fv := parsed.val.Field(p)
			fv.Set(reflect.ValueOf(posArgs))
			posArgs = nil
		} else {
			a := posArgs[0]
			posArgs = posArgs[1:]
			err := parsed.setSwitch(p, a)
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
