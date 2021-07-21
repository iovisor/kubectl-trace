package errors

const (
	ErrorInvalid       string = "Invalid"
	ErrorUnallocatable string = "Unallocatable"
)

type TargetSelectionError struct {
	ErrorMessage string
	ErrorType    string
}

func (e TargetSelectionError) Error() string {
	return e.ErrorMessage
}

func NewErrorInvalid(message string) error {
	return TargetSelectionError{
		ErrorMessage: message,
		ErrorType:    ErrorInvalid,
	}
}

func NewErrorUnallocatable(message string) error {
	return TargetSelectionError{
		ErrorMessage: message,
		ErrorType:    ErrorUnallocatable,
	}
}

func IsInvalidTargetError(err error) bool {
	return reasonForError(err) == ErrorInvalid
}

func IsUnallocatableTargetError(err error) bool {
	return reasonForError(err) == ErrorUnallocatable
}

func reasonForError(err error) string {
	myErr, ok := err.(TargetSelectionError)
	if ok {
		return myErr.ErrorType
	}
	return ""
}
