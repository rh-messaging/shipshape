package validation

import (
	"github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/qdrmanagement/entities"
	"github.com/onsi/gomega"
	"reflect"
)

// ValidateEntityValues uses reflect to compare values from a given entity's field
// with the provided value from fieldValues map.
//
// This way you do not need to compare the whole entity, but just the fields that
// are relevant to match.
func ValidateEntityValues(entity entities.Entity, fieldValues map[string]interface{}) {
	element := reflect.Indirect(reflect.ValueOf(entity))
	for field, fieldValue := range fieldValues {
		currentValue := element.FieldByName(field).Interface()
		gomega.Expect(currentValue).To(gomega.Equal(fieldValue))
	}
}
