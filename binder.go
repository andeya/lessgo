package lessgo

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type (
	// Binder is the interface that wraps the Bind method.
	Binder interface {
		Bind(interface{}, *Context) error
	}

	binder struct{}
)

const (
	bindStructTag  = "bind"
	bindStructTag2 = "json"
)

func (b *binder) Bind(i interface{}, c *Context) error {
	req := c.request
	ctype := req.Header.Get(HeaderContentType)
	if req.Body == nil {
		return NewHTTPError(http.StatusBadRequest, "request body can't be empty")
	}
	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		if err := json.NewDecoder(req.Body).Decode(i); err != nil {
			return NewHTTPError(http.StatusBadRequest, err.Error())
		}
	case strings.HasPrefix(ctype, MIMEApplicationXML):
		if err := xml.NewDecoder(req.Body).Decode(i); err != nil {
			return NewHTTPError(http.StatusBadRequest, err.Error())
		}
	case strings.HasPrefix(ctype, MIMEApplicationForm), strings.HasPrefix(ctype, MIMEMultipartForm):
		typ := reflect.TypeOf(i)
		if typ.Kind() != reflect.Ptr {
			return NewHTTPError(http.StatusBadRequest, "When \"Content-Type: "+ctype+"\", \"Bind()\"'s param must be \"*struct\".")
		}
		typ = typ.Elem()
		if typ.Kind() != reflect.Struct {
			return NewHTTPError(http.StatusBadRequest, "When \"Content-Type: "+ctype+"\", \"Bind()\"'s param must be \"*struct\".")
		}
		val := reflect.ValueOf(i).Elem()
		if err := b.bindForm(typ, val, c.FormValues()); err != nil {
			return NewHTTPError(http.StatusBadRequest, err.Error())
		}
	default:
		return ErrUnsupportedMediaType
	}
	return nil
}

func (b *binder) bindForm(typ reflect.Type, val reflect.Value, form url.Values) error {
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		var inputFieldName string
		inputFieldName = strings.TrimSpace(typeField.Tag.Get(bindStructTag))
		if inputFieldName == "" {
			inputFieldName = strings.TrimSpace(typeField.Tag.Get(bindStructTag2))
		}
		if inputFieldName == "-" {
			continue
		}
		if inputFieldName == "" {
			inputFieldName = typeField.Name
			// If bindStructTag or bindStructTag2 tag is null, we inspect if the field is a struct or *struct.
			if structFieldKind == reflect.Ptr {
				structField = structField.Elem()
			}
			if structFieldKind == reflect.Struct {
				err := b.bindForm(structField.Type(), structField, form)
				if err != nil {
					return err
				}
				continue
			}
		}
		inputFieldName = strings.TrimSpace(strings.Split(inputFieldName, ",")[0])
		inputValue, exists := form[inputFieldName]
		if !exists {
			continue
		}

		numElems := len(inputValue)
		if structFieldKind == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for i := 0; i < numElems; i++ {
				if err := setWithProperType(sliceOf, inputValue[i], slice.Index(i)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else {
			if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
				return err
			}
		}
	}
	return nil
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	switch valueKind {
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	default:
		return errors.New("unknown type")
	}
	return nil
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}
