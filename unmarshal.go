package up

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshal parses UP document and stores the result in the value pointed to by v.
// If v is not a pointer to a struct, Unmarshal returns an error.
//
// Unmarshal uses struct tags to determine how to map UP keys to struct fields:
//   - `up:"fieldname"` - maps UP key "fieldname" to this struct field
//   - `up:"fieldname,omitempty"` - omits field if value is empty
//   - `up:"-"` - ignores this field
//
// Example:
//
//	type Config struct {
//	    Host     string `up:"host"`
//	    Port     int    `up:"port"`
//	    Enabled  bool   `up:"enabled"`
//	    Tags     []string `up:"tags"`
//	    Database struct {
//	        Host string `up:"host"`
//	        Port int    `up:"port"`
//	    } `up:"database"`
//	}
func Unmarshal(data []byte, v any) error {
	parser := NewParser()
	doc, err := parser.ParseDocument(strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	return UnmarshalDocument(doc, v)
}

// UnmarshalDocument unmarshals a parsed Document into v.
func UnmarshalDocument(doc *Document, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("unmarshal target must be a non-nil pointer")
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal target must be a pointer to struct")
	}

	// Create a map from document nodes
	data := make(map[string]any)
	for _, node := range doc.Nodes {
		data[node.Key] = node.Value
	}

	return unmarshalStruct(data, elem)
}

// unmarshalStruct unmarshals a map into a struct value
func unmarshalStruct(data map[string]any, v reflect.Value) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get UP tag
		tag := field.Tag.Get("up")
		if tag == "-" {
			continue
		}

		// Parse tag options
		tagName, opts := parseTag(tag)
		if tagName == "" {
			// Use field name as default
			tagName = strings.ToLower(field.Name)
		}

		// Get value from data
		value, ok := data[tagName]
		if !ok {
			if hasOption(opts, "required") {
				return fmt.Errorf("required field %s not found", tagName)
			}
			continue
		}

		// Check omitempty
		if hasOption(opts, "omitempty") && isEmpty(value) {
			continue
		}

		// Set the field value
		if err := setField(fieldValue, value); err != nil {
			return fmt.Errorf("field %s: %v", field.Name, err)
		}
	}

	return nil
}

// setField sets a reflect.Value based on the any value
func setField(field reflect.Value, value any) error {
	if value == nil {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		return setString(field, value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setInt(field, value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setUint(field, value)
	case reflect.Float32, reflect.Float64:
		return setFloat(field, value)
	case reflect.Bool:
		return setBool(field, value)
	case reflect.Slice:
		return setSlice(field, value)
	case reflect.Map:
		return setMap(field, value)
	case reflect.Struct:
		return setStruct(field, value)
	case reflect.Ptr:
		return setPointer(field, value)
	case reflect.Interface:
		field.Set(reflect.ValueOf(value))
		return nil
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
}

func setString(field reflect.Value, value any) error {
	switch v := value.(type) {
	case string:
		field.SetString(v)
	default:
		field.SetString(fmt.Sprint(v))
	}
	return nil
}

func setInt(field reflect.Value, value any) error {
	switch v := value.(type) {
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse as int: %v", err)
		}
		field.SetInt(i)
	case int:
		field.SetInt(int64(v))
	case int64:
		field.SetInt(v)
	case float64:
		field.SetInt(int64(v))
	default:
		return fmt.Errorf("cannot convert %T to int", v)
	}
	return nil
}

func setUint(field reflect.Value, value any) error {
	switch v := value.(type) {
	case string:
		i, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse as uint: %v", err)
		}
		field.SetUint(i)
	case int:
		field.SetUint(uint64(v))
	case int64:
		field.SetUint(uint64(v))
	case uint64:
		field.SetUint(v)
	case float64:
		field.SetUint(uint64(v))
	default:
		return fmt.Errorf("cannot convert %T to uint", v)
	}
	return nil
}

func setFloat(field reflect.Value, value any) error {
	switch v := value.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("cannot parse as float: %v", err)
		}
		field.SetFloat(f)
	case int:
		field.SetFloat(float64(v))
	case int64:
		field.SetFloat(float64(v))
	case float64:
		field.SetFloat(v)
	default:
		return fmt.Errorf("cannot convert %T to float", v)
	}
	return nil
}

func setBool(field reflect.Value, value any) error {
	switch v := value.(type) {
	case string:
		b, err := parseBool(v)
		if err != nil {
			return fmt.Errorf("cannot parse as bool: %v", err)
		}
		field.SetBool(b)
	case bool:
		field.SetBool(v)
	default:
		return fmt.Errorf("cannot convert %T to bool", v)
	}
	return nil
}

func setSlice(field reflect.Value, value any) error {
	switch v := value.(type) {
	case List:
		slice := reflect.MakeSlice(field.Type(), len(v), len(v))
		for i, item := range v {
			if err := setField(slice.Index(i), item); err != nil {
				return fmt.Errorf("index %d: %v", i, err)
			}
		}
		field.Set(slice)
	case []any:
		slice := reflect.MakeSlice(field.Type(), len(v), len(v))
		for i, item := range v {
			if err := setField(slice.Index(i), item); err != nil {
				return fmt.Errorf("index %d: %v", i, err)
			}
		}
		field.Set(slice)
	default:
		return fmt.Errorf("cannot convert %T to slice", v)
	}
	return nil
}

func setMap(field reflect.Value, value any) error {
	switch v := value.(type) {
	case Block:
		m := reflect.MakeMap(field.Type())
		for key, val := range v {
			keyValue := reflect.ValueOf(key)
			elemValue := reflect.New(field.Type().Elem()).Elem()
			if err := setField(elemValue, val); err != nil {
				return fmt.Errorf("key %s: %v", key, err)
			}
			m.SetMapIndex(keyValue, elemValue)
		}
		field.Set(m)
	case map[string]any:
		m := reflect.MakeMap(field.Type())
		for key, val := range v {
			keyValue := reflect.ValueOf(key)
			elemValue := reflect.New(field.Type().Elem()).Elem()
			if err := setField(elemValue, val); err != nil {
				return fmt.Errorf("key %s: %v", key, err)
			}
			m.SetMapIndex(keyValue, elemValue)
		}
		field.Set(m)
	default:
		return fmt.Errorf("cannot convert %T to map", v)
	}
	return nil
}

func setStruct(field reflect.Value, value any) error {
	switch v := value.(type) {
	case Block:
		// Block is map[string]Value, convert to map[string]any
		m := make(map[string]any)
		for k, val := range v {
			m[k] = val
		}
		return unmarshalStruct(m, field)
	case map[string]any:
		return unmarshalStruct(v, field)
	default:
		return fmt.Errorf("cannot convert %T to struct", v)
	}
}

func setPointer(field reflect.Value, value any) error {
	if value == nil {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	// Create new pointer
	ptr := reflect.New(field.Type().Elem())
	if err := setField(ptr.Elem(), value); err != nil {
		return err
	}
	field.Set(ptr)
	return nil
}

// Helper functions

func parseTag(tag string) (string, []string) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

func hasOption(opts []string, option string) bool {
	for _, opt := range opts {
		if opt == option {
			return true
		}
	}
	return false
}

func isEmpty(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	default:
		return false
	}
}

func parseBool(s string) (bool, error) {
	s = strings.ToLower(s)
	switch s {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value: %s", s)
	}
}

