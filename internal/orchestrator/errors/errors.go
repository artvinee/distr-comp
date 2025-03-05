package agent

import (
	"errors"
)

var (
	ErrInvalidExpression     = errors.New("invalid expression")
	ErrMismatchedParentheses = errors.New("mismatched parentheses")
	ErrInvalidNumber         = errors.New("invalid number")
	ErrDivisionByZero        = errors.New("division by zero")

	ErrTaskNotFound      = errors.New("task not found")
	ErrInvalidTaskResult = errors.New("invalid task result")
	ErrNoTasksAvailable  = errors.New("no tasks available")
)
