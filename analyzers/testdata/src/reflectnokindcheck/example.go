package reflectnokindcheck

import "reflect"

// BadFieldsNoCheck calls Fields without Kind check.
func BadFieldsNoCheck(t reflect.Type) {
	for f := range t.Fields() { // want `reflect\.Type\.Fields\(\) panics if Kind is not Struct`
		_ = f
	}
}

// BadNumFieldNoCheck calls NumField without Kind check.
func BadNumFieldNoCheck(t reflect.Type) {
	_ = t.NumField() // want `reflect\.Type\.NumField\(\) panics if Kind is not Struct`
}

// GoodFieldsWithIfGuard has Kind check in enclosing if.
func GoodFieldsWithIfGuard(t reflect.Type) {
	if t.Kind() == reflect.Struct {
		for f := range t.Fields() {
			_ = f
		}
	}
}

// GoodNumFieldWithIfGuard has Kind check in enclosing if.
func GoodNumFieldWithIfGuard(t reflect.Type) {
	if t.Kind() == reflect.Struct {
		_ = t.NumField()
	}
}

// GoodFieldsInSwitch has Kind check via switch.
func GoodFieldsInSwitch(t reflect.Type) {
	switch t.Kind() {
	case reflect.Struct:
		for f := range t.Fields() {
			_ = f
		}
	}
}

// GoodNumFieldInSwitch has Kind check via switch on NumField.
func GoodNumFieldInSwitch(t reflect.Type) {
	switch t.Kind() {
	case reflect.Struct:
		_ = t.NumField()
	}
}
