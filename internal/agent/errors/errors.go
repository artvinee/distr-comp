package agent

import (
	"errors"
)

var (
	ErrInvalidExpression     = errors.New("invalid expression")
	ErrMismatchedParentheses = errors.New("mismatched parentheses")
	ErrInvalidNumber         = errors.New("invalid number")
	ErrDivisionByZero        = errors.New("division by zero")
)
