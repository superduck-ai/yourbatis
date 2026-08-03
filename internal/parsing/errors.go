package parsing

type ParsingException struct {
	message string
	cause   error
}

func NewParsingException(message string, cause error) *ParsingException {
	return &ParsingException{
		message: message,
		cause:   cause,
	}
}

func (exception *ParsingException) Error() string {
	if exception == nil {
		return ""
	}
	if exception.message != "" {
		return exception.message
	}
	if exception.cause != nil {
		return exception.cause.Error()
	}
	return ""
}

func (exception *ParsingException) Unwrap() error {
	if exception == nil {
		return nil
	}
	return exception.cause
}
