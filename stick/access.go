package stick

import (
	"fmt"
	"reflect"
)

// Field is dynamically accessible field.
type Field struct {
	Index int
	Type  reflect.Type
}

// Accessor provides dynamic access to a structs fields.
type Accessor struct {
	Name   string
	Fields map[string]*Field
}

// Accessible is a type that has dynamically accessible fields.
type Accessible interface {
	GetAccessor(interface{}) *Accessor
}

// BuildAccessor will build an accessor for the provided type.
func BuildAccessor(v interface{}, ignore ...string) *Accessor {
	// get type
	typ := reflect.TypeOf(v)

	// unwrap pointer
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// prepare accessor
	accessor := &Accessor{
		Name:   typ.String(),
		Fields: map[string]*Field{},
	}

	// add fields
	// iterate through all fields
	for i := 0; i < typ.NumField(); i++ {
		// get field
		field := typ.Field(i)

		// check field
		var skip bool
		for _, ign := range ignore {
			if ign == field.Name {
				skip = true
			}
		}
		if skip {
			continue
		}

		// add field
		accessor.Fields[field.Name] = &Field{
			Index: i,
			Type:  field.Type,
		}
	}

	return accessor
}

// Get will lookup the specified field on the accessible and return its value
// and whether the field was found at all.
func Get(acc Accessible, name string) (interface{}, bool) {
	// find field
	field := acc.GetAccessor(acc).Fields[name]
	if field == nil {
		return nil, false
	}

	// get value
	value := reflect.ValueOf(acc).Elem().Field(field.Index).Interface()

	return value, true
}

// Set will set the specified field on the accessible with the provided value
// and return whether the field has been found and the value has been set.
func Set(acc Accessible, name string, value interface{}) bool {
	// find field
	field := acc.GetAccessor(acc).Fields[name]
	if field == nil {
		return false
	}

	// get value
	fieldValue := reflect.ValueOf(acc).Elem().Field(field.Index)

	// get value value
	valueValue := reflect.ValueOf(value)

	// correct untyped nil values
	if fieldValue.Type().Kind() == reflect.Ptr && value == nil {
		valueValue = reflect.Zero(fieldValue.Type())
	}

	// check type
	if fieldValue.Type() != valueValue.Type() {
		return false
	}

	// set value
	fieldValue.Set(valueValue)

	return true
}

// MustGet will call Get and panic if the operation failed.
func MustGet(acc Accessible, name string) interface{} {
	// get value
	value, ok := Get(acc, name)
	if !ok {
		panic(fmt.Sprintf(`stick: could not get field "%s" on "%s"`, name, acc.GetAccessor(acc).Name))
	}

	return value
}

// MustSet will call Set and panic if the operation failed.
func MustSet(acc Accessible, name string, value interface{}) {
	// get value
	ok := Set(acc, name, value)
	if !ok {
		panic(fmt.Sprintf(`stick: could not set "%s" on "%s"`, name, acc.GetAccessor(acc).Name))
	}
}
