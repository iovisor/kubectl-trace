package meta

import (
	"strings"
)

// IsObjectName return true if the provived string
func IsObjectName(name string) bool {
	if strings.Compare(ObjectNamePrefix, name) == 0 {
		return false
	}
	return strings.HasPrefix(name, ObjectNamePrefix)
}
