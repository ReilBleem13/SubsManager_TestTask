package domain

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) WithMessage(msg string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: msg,
		Status:  e.Status,
	}
}

func ErrInternalServer() *AppError {
	return &AppError{
		Code:    "INTERNAL_SERVER",
		Message: "Internal server error",
		Status:  500,
	}
}

func ErrBadRequest() *AppError {
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: "Bad request",
		Status:  400,
	}
}

func ErrNotFound() *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: "Object not found",
		Status:  404,
	}
}

func ErrAlreadyExist() *AppError {
	return &AppError{
		Code:    "ALREADY_EXISTS",
		Message: "Object already exists",
		Status:  409,
	}
}
