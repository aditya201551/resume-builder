package design

import "testing"

func TestSchemaEnumFieldsMatchDefaultDesignValues(t *testing.T) {
	d := Default("template-id")
	if err := Validate(d); err != nil {
		t.Fatalf("Default() must be valid for this test to be meaningful: %v", err)
	}

	for _, group := range Schema() {
		for _, field := range group.Fields {
			if field.Type != FieldEnum {
				continue
			}
			if len(field.Options) == 0 {
				t.Errorf("field %q is FieldEnum but has no Options", field.Key)
			}
		}
	}
}

func TestSchemaNumberFieldsHaveRange(t *testing.T) {
	for _, group := range Schema() {
		for _, field := range group.Fields {
			if field.Type != FieldNumber {
				continue
			}
			if field.Min == nil || field.Max == nil || field.Step == nil {
				t.Errorf("field %q is FieldNumber but missing Min/Max/Step", field.Key)
			} else if *field.Min >= *field.Max {
				t.Errorf("field %q has Min >= Max (%v >= %v)", field.Key, *field.Min, *field.Max)
			}
		}
	}
}
