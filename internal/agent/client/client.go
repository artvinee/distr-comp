package agent

import (
	"bytes"
	"distr-comp/internal/logger"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func Start(computingPower int, orchestratorURL string) {
	logger.Infof("Starting agent with orchestrator URL: %s", orchestratorURL)
	agent := NewAgent(orchestratorURL)

	var wg sync.WaitGroup
	wg.Add(computingPower)

	for i := 0; i < computingPower; i++ {
		go func(workerID int) {
			logger.Infof("Starting worker #%d", workerID)
			defer wg.Done()

			for {
				task, err := agent.GetTask()
				if err != nil {
					logger.Errorf("Worker #%d: Failed to get task: %v", workerID, err)
					time.Sleep(time.Millisecond * 200)
					continue
				} else if task == nil {
					time.Sleep(time.Millisecond * 350)
					continue
				}

				logger.Infof("Worker #%d: Processing task %s: %s %v %v", workerID, task.ID, task.Operation, task.Arg1, task.Arg2)
				result, err := SolveTask(task)
				if err != nil {
					logger.Errorf("Worker #%d: Failed to solve task %s: %v", workerID, task.ID, err)
					time.Sleep(time.Millisecond * 200)
					continue
				}

				logger.Infof("Worker #%d: Completed task %s with result %v", workerID, task.ID, result.Result)
				if err := agent.SubmitResult(result); err != nil {
					logger.Errorf("Worker #%d: Failed to submit result for task %s: %v", workerID, task.ID, err)
				}
			}
		}(i)
	}

	wg.Wait()
}

func NewAgent(orchestratorURL string) *Agent {
	logger.Infof("Creating new agent with orchestrator URL: %s", orchestratorURL)
	return &Agent{orchestratorURL: orchestratorURL, client: &http.Client{}}
}

var agentServerOffline bool

func (a *Agent) GetTask() (*Task, error) {
	resp, err := a.client.Get(a.orchestratorURL + "/internal/task")
	if err != nil {
		if !agentServerOffline {
			logger.Warnf("Failed to connect to server at %s. Will retry", a.orchestratorURL)
			agentServerOffline = true
		}
		return nil, err
	}
	defer resp.Body.Close()

	agentServerOffline = false

	if resp.StatusCode == http.StatusNotFound {
		logger.Debug("No tasks available")
		return nil, nil
	} else if resp.StatusCode != http.StatusOK {
		logger.Errorf("Unexpected status code: %d", resp.StatusCode)
		return nil, errors.New("failed to get task")
	}

	var respBody struct {
		Task *Task `json:"task"`
	}

	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		logger.Errorf("Error decoding task JSON: %v", err)
		return nil, err
	}

	jsonTask, err := json.MarshalIndent(respBody.Task, "", "  ")
	if err != nil {
		logger.Errorf("Error marshalling task to JSON: %v", err)
		return nil, err
	}

	logger.Infof("Received task: %s", string(jsonTask))
	return respBody.Task, nil
}

func (a *Agent) SubmitResult(result *TaskResultRequest) error {
	jsonResult, err := json.Marshal(result)
	if err != nil {
		logger.Errorf("Error marshalling result to JSON: %v", err)
		return err
	}

	resp, err := a.client.Post(a.orchestratorURL+"/internal/task", "application/json", bytes.NewBuffer(jsonResult))
	if err != nil {
		logger.Errorf("Error submitting result: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("Unexpected status code when submitting result: %d", resp.StatusCode)
		return fmt.Errorf("failed to submit result, status code: %d", resp.StatusCode)
	}

	logger.Infof("Successfully submitted result for task %s: %v", result.ID, result.Result)
	return nil
}

func getOperationTime(task *Task) time.Duration {
	if task.OperationTime > 0 {
		return time.Duration(task.OperationTime) * time.Millisecond
	}

	var envVar string
	switch task.Operation {
	case "+":
		envVar = "TIME_ADDITION_MS"
	case "-":
		envVar = "TIME_SUBTRACTION_MS"
	case "*":
		envVar = "TIME_MULTIPLICATIONS_MS"
	case "/":
		envVar = "TIME_DIVISIONS_MS"
	default:
		return 0
	}

	if timeStr := os.Getenv(envVar); timeStr != "" {
		if timeMs, err := strconv.Atoi(timeStr); err == nil {
			return time.Duration(timeMs) * time.Millisecond
		}
	}

	switch task.Operation {
	case "+":
		return 100 * time.Millisecond
	case "-":
		return 100 * time.Millisecond
	case "*":
		return 200 * time.Millisecond
	case "/":
		return 300 * time.Millisecond
	default:
		return 0
	}
}

func SolveTask(task *Task) (*TaskResultRequest, error) {
	var num1, num2 float64
	var err error

	num1, err = convertToFloat(task.Arg1)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Arg1: %v", err)
	}

	num2, err = convertToFloat(task.Arg2)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Arg2: %v", err)
	}

	logger.Debugf("Solving task: %v %s %v", num1, task.Operation, num2)

	operationTime := getOperationTime(task)
	logger.Debugf("Operation %s will take %v", task.Operation, operationTime)

	var result float64
	switch task.Operation {
	case "+":
		time.Sleep(operationTime)
		result = num1 + num2
	case "-":
		time.Sleep(operationTime)
		result = num1 - num2
	case "*":
		time.Sleep(operationTime)
		result = num1 * num2
	case "/":
		if num2 == 0 {
			logger.Errorf("Division by zero in task %s", task.ID)
			return nil, errors.New("division by zero")
		}
		time.Sleep(operationTime)
		result = num1 / num2
	default:
		logger.Errorf("Unsupported operation in task %s: %s", task.ID, task.Operation)
		return nil, errors.New("unsupported operation")
	}

	logger.Debugf("Task %s result: %v", task.ID, result)
	return &TaskResultRequest{ID: task.ID, Result: result}, nil
}

func convertToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, errors.New("unsupported type")
	}
}
