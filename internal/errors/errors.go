package errors

import "strings"

type MultiError []error

func (e MultiError) Error() string {
	if len(e) == 0 {
		return ""
	}
	msg := make([]string, 0, len(e))
	for _, err := range e {
		msg = append(msg, err.Error())
	}
	return strings.Join(msg, ",")
}
