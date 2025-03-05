package agent

import "net/http"

type Agent struct {
	orchestratorURL string
	client          *http.Client
}

type Task struct {
	ID            string      `json:"id"`
	Arg1          interface{} `json:"arg1"`
	Arg2          interface{} `json:"arg2"`
	Operation     string      `json:"operation"`
	OperationTime int         `json:"operation_time"`
}

type TaskResultRequest struct {
	ID     string  `json:"id"`
	Result float64 `json:"result"`
}
