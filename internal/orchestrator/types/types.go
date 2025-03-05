package orchestrator

import (
	"sync"
)

type Task struct {
	ID            string   `json:"id"`
	ExpressionID  string   `json:"expression_id"`
	Arg1          string   `json:"arg1"`
	Arg2          string   `json:"arg2"`
	Operation     string   `json:"operation"`
	OperationTime int      `json:"operation_time"`
	Dependencies  []string `json:"-"`
	Status        string   `json:"status"`
	Result        *float64 `json:"result"`
}

type Token struct {
	Type    TokenType
	Value   string
	IsUnary bool
}

type Expression struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Result *float64 `json:"result"`
	Tasks  []*Task  `json:"-"`
	mu     sync.Mutex
}

type TokenType int

type ExpressionResponse struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Result *float64 `json:"result,omitempty"`
}

type TaskResponse struct {
	ID            string  `json:"id"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Operation     string  `json:"operation"`
	OperationTime int     `json:"operation_time"`
}
