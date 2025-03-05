package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"
	"unicode"

	logger "distr-comp/internal/logger"
	errs "distr-comp/internal/orchestrator/errors"
	types "distr-comp/internal/orchestrator/types"
	utils "distr-comp/internal/orchestrator/utils"
)

const (
	TokenNumber types.TokenType = iota
	TokenOperator
	TokenLeftParen
	TokenRightParen

	StatusPending  = "pending"
	StatusReady    = "ready"
	StatusProgress = "in_progress"
	StatusDone     = "done"
	StatusError    = "error"
)

type Orchestrator struct {
	Expressions       map[string]*types.Expression
	Tasks             map[string]*types.Task
	ReadyTasks        chan *types.Task
	ProcessingTasks   map[string]bool
	OperationTimes    map[string]time.Duration
	ComputingPower    int
	Mu                sync.RWMutex
	ExpressionCounter int
	TaskCounter       int
}

func NewOrchestrator(timeAddition, timeSubtraction, timeMultiplication, timeDivision time.Duration) *Orchestrator {
	logger.Infof("Initializing new orchestrator with operation times: +=%v, -=%v, *=%v, /=%v",
		timeAddition, timeSubtraction, timeMultiplication, timeDivision)

	return &Orchestrator{
		Expressions:     make(map[string]*types.Expression),
		Tasks:           make(map[string]*types.Task),
		ReadyTasks:      make(chan *types.Task, 8192),
		ProcessingTasks: make(map[string]bool),
		OperationTimes: map[string]time.Duration{
			"+": timeAddition,
			"-": timeSubtraction,
			"*": timeMultiplication,
			"/": timeDivision,
		},
	}
}

func (o *Orchestrator) AddExpression(expr string) (string, error) {
	o.Mu.Lock()
	defer o.Mu.Unlock()

	o.ExpressionCounter++
	exprID := fmt.Sprintf("expr-%d", o.ExpressionCounter)

	tokens, err := tokenize(expr)
	if err != nil {
		return "", err
	}
	rpn, err := toRPN(tokens)
	if err != nil {
		return "", err
	}

	expression := &types.Expression{
		ID:     exprID,
		Status: StatusPending,
		Tasks:  o.parseExpression(rpn, exprID),
	}

	for _, task := range expression.Tasks {
		o.Tasks[task.ID] = task
		if len(task.Dependencies) == 0 {
			task.Status = StatusReady
			go func(t *types.Task) {
				o.ReadyTasks <- t
			}(task)
		}
	}

	o.Expressions[exprID] = expression
	return exprID, nil
}

func (o *Orchestrator) parseExpression(tokens []types.Token, exprID string) []*types.Task {
	var stack []string
	var tasks []*types.Task

	for _, token := range tokens {
		switch token.Type {
		case TokenNumber:
			stack = append(stack, token.Value)
		case TokenOperator:
			if len(stack) < 2 {
				continue
			}
			arg2 := stack[len(stack)-1]
			arg1 := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			o.TaskCounter++
			taskID := fmt.Sprintf("task-%d", o.TaskCounter)

			deps := make([]string, 0)
			if !utils.IsNumber(arg1) {
				deps = append(deps, arg1)
			}
			if !utils.IsNumber(arg2) {
				deps = append(deps, arg2)
			}

			task := &types.Task{
				ID:           taskID,
				ExpressionID: exprID,
				Arg1:         arg1,
				Arg2:         arg2,
				Operation:    token.Value,
				Dependencies: deps,
				Status:       StatusPending,
			}

			tasks = append(tasks, task)
			stack = append(stack, taskID)
		}
	}
	taskJSON, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling task to JSON:", err)
	}
	fmt.Println(string(taskJSON))
	return tasks
}

func (o *Orchestrator) GetNextTask() (*types.Task, error) {
	o.Mu.Lock()
	defer o.Mu.Unlock()

	select {
	case task := <-o.ReadyTasks:
		if task.Status == StatusReady {
			task.Status = StatusProgress
			o.ProcessingTasks[task.ID] = true
			return task, nil
		}
	default:
		return nil, errs.ErrNoTasksAvailable
	}
	return nil, errs.ErrNoTasksAvailable
}

func (o *Orchestrator) ProcessTaskResult(taskID string, result float64) error {
	o.Mu.Lock()
	defer o.Mu.Unlock()

	task, exists := o.Tasks[taskID]
	if !exists || task.Status != StatusProgress {
		return errors.New("invalid task")
	}

	task.Status = StatusDone
	task.Result = &result
	delete(o.ProcessingTasks, taskID)

	for _, t := range o.Tasks {
		if utils.Contains(t.Dependencies, taskID) {
			t.Dependencies = utils.Remove(t.Dependencies, taskID)
			if len(t.Dependencies) == 0 && t.Status == StatusPending {
				t.Status = StatusReady
				o.ReadyTasks <- t
			}
		}
	}

	if expr, exists := o.Expressions[task.ExpressionID]; exists {
		allDone := true
		for _, t := range expr.Tasks {
			if t.Status != StatusDone {
				allDone = false
				break
			}
		}
		if allDone {
			expr.Status = StatusDone
			expr.Result = task.Result
		}
	}

	return nil
}

func tokenize(expression string) ([]types.Token, error) {
	var tokens []types.Token
	var i int
	var prevToken types.Token

	for i < len(expression) {
		ch := expression[i]

		if unicode.IsSpace(rune(ch)) {
			i++
			continue
		}

		if ch == '(' {
			tokens = append(tokens, types.Token{Type: TokenLeftParen, Value: string(ch)})
			prevToken = tokens[len(tokens)-1]
			i++
			continue
		}

		if ch == ')' {
			tokens = append(tokens, types.Token{Type: TokenRightParen, Value: string(ch)})
			prevToken = tokens[len(tokens)-1]
			i++
			continue
		}

		if ch == '+' || ch == '-' || ch == '*' || ch == '/' {
			isUnary := false
			if ch == '+' || ch == '-' {
				if len(tokens) == 0 || prevToken.Type == TokenOperator || prevToken.Type == TokenLeftParen {
					isUnary = true
				}
			}

			if isUnary && i+1 < len(expression) && expression[i+1] == ' ' {
				return nil, fmt.Errorf("unary operator '%c' must be directly before the number or '(', without spaces", ch)
			}

			if !isUnary && prevToken.Type == TokenOperator && !prevToken.IsUnary {
				return nil, fmt.Errorf("two operators '%s' and '%s' cannot be next to each other at position %d", prevToken.Value, string(ch), i)
			}

			tokens = append(tokens, types.Token{Type: TokenOperator, Value: string(ch), IsUnary: isUnary})
			prevToken = tokens[len(tokens)-1]
			i++
			continue
		}

		if unicode.IsDigit(rune(ch)) || ch == '.' {
			start := i
			dotCount := 0
			for i < len(expression) && (unicode.IsDigit(rune(expression[i])) || expression[i] == '.') {
				if expression[i] == '.' {
					dotCount++
					if dotCount > 1 {
						return nil, fmt.Errorf("invalid number format with multiple dots at position %d", i)
					}
				}
				i++
			}
			numberStr := expression[start:i]
			tokens = append(tokens, types.Token{Type: TokenNumber, Value: numberStr})
			prevToken = tokens[len(tokens)-1]
			continue
		}

		return nil, fmt.Errorf("unknown character '%c' at position %d", ch, i)
	}
	return tokens, nil
}

func ValidateExpression(expression string) error {
	if len(expression) == 0 {
		return fmt.Errorf("expression cannot be empty")
	}

	parenCount := 0

	for i := 0; i < len(expression); i++ {
		ch := expression[i]

		if ch == '(' {
			parenCount++
		} else if ch == ')' {
			parenCount--
			if parenCount < 0 {
				return fmt.Errorf("unbalanced parentheses at position %d", i)
			}
		}

		if !unicode.IsDigit(rune(ch)) && ch != '.' && ch != '+' && ch != '-' &&
			ch != '*' && ch != '/' && ch != '(' && ch != ')' && !unicode.IsSpace(rune(ch)) {
			return fmt.Errorf("invalid character '%c' at position %d", ch, i)
		}

		if i == len(expression)-1 && (ch == '+' || ch == '-' || ch == '*' || ch == '/') {
			return fmt.Errorf("expression cannot end with operator '%c'", ch)
		}

		if i > 0 && (ch == '*' || ch == '/') &&
			(expression[i-1] == '+' || expression[i-1] == '-' ||
				expression[i-1] == '*' || expression[i-1] == '/') {
			return fmt.Errorf("two operators '%c' and '%c' cannot be adjacent at position %d", expression[i-1], ch, i)
		}
	}

	if parenCount > 0 {
		return fmt.Errorf("unbalanced parentheses: %d closing parentheses are missing", parenCount)
	}

	return nil
}

func (o *Orchestrator) GetAllExpressions() ([]*types.Expression, error) {
	o.Mu.RLock()
	defer o.Mu.RUnlock()

	if o.Expressions == nil {
		return nil, fmt.Errorf("expressions map is nil")
	}

	expressions := make([]*types.Expression, 0, len(o.Expressions))
	for _, expr := range o.Expressions {
		if expr == nil {
			return nil, fmt.Errorf("nil expression found in map")
		}
		expressions = append(expressions, expr)
	}
	return expressions, nil
}

func (o *Orchestrator) GetExpression(id string) (*types.Expression, bool, error) {
	o.Mu.RLock()
	defer o.Mu.RUnlock()

	if o.Expressions == nil {
		return nil, false, fmt.Errorf("expressions map is nil")
	}

	expr, exists := o.Expressions[id]
	if !exists {
		return nil, false, nil
	}

	if expr == nil {
		return nil, true, fmt.Errorf("nil expression found in map for id %s", id)
	}

	return expr, true, nil
}

func (o *Orchestrator) ResolveTaskDependencies(task *types.Task) map[string]interface{} {
	result := make(map[string]interface{})
	o.Mu.RLock()
	defer o.Mu.RUnlock()

	resolveArg := func(arg string) float64 {
		if utils.IsNumber(arg) {
			val, _ := strconv.ParseFloat(arg, 64)
			return val
		}
		if t, exists := o.Tasks[arg]; exists && t.Result != nil {
			return *t.Result
		}
		return math.NaN()
	}

	arg1 := resolveArg(task.Arg1)
	arg2 := resolveArg(task.Arg2)

	result["id"] = task.ID
	result["operation"] = task.Operation
	result["arg1"] = arg1
	result["arg2"] = arg2
	result["operation_time"] = o.OperationTimes[task.Operation]

	return result
}

func toRPN(tokens []types.Token) ([]types.Token, error) {
	var outputQueue []types.Token
	var operatorStack []types.Token

	for _, token := range tokens {
		switch token.Type {
		case TokenNumber:
			outputQueue = append(outputQueue, token)
		case TokenOperator:
			if token.IsUnary {
				operatorStack = append(operatorStack, token)
				continue
			}
			for len(operatorStack) > 0 {
				top := operatorStack[len(operatorStack)-1]
				if top.Type == TokenOperator && operatorPrecedence(top.Value) >= operatorPrecedence(token.Value) {
					outputQueue = append(outputQueue, top)
					operatorStack = operatorStack[:len(operatorStack)-1]
				} else {
					break
				}
			}
			operatorStack = append(operatorStack, token)
		case TokenLeftParen:
			operatorStack = append(operatorStack, token)
		case TokenRightParen:
			foundLeftParen := false
			for len(operatorStack) > 0 {
				top := operatorStack[len(operatorStack)-1]
				operatorStack = operatorStack[:len(operatorStack)-1]
				if top.Type == TokenLeftParen {
					foundLeftParen = true
					break
				} else {
					outputQueue = append(outputQueue, top)
				}
			}
			if !foundLeftParen {
				return nil, fmt.Errorf("mismatched parentheses")
			}
		}
	}

	for len(operatorStack) > 0 {
		top := operatorStack[len(operatorStack)-1]
		if top.Type == TokenLeftParen || top.Type == TokenRightParen {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		outputQueue = append(outputQueue, top)
		operatorStack = operatorStack[:len(operatorStack)-1]
	}

	return outputQueue, nil
}

func operatorPrecedence(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	default:
		return 0
	}
}
