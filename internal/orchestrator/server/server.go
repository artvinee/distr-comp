package orchestrator

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	logger "distr-comp/internal/logger"
	core "distr-comp/internal/orchestrator/core"
	errs "distr-comp/internal/orchestrator/errors"
	types "distr-comp/internal/orchestrator/types"
	utils "distr-comp/internal/orchestrator/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	Engine       *gin.Engine
	Orchestrator *core.Orchestrator
}

func NewServer(timeAddition, timeSubtraction, timeMultiplication, timeDivision time.Duration) *Server {
	engine := gin.Default()
	server := &Server{
		Engine:       engine,
		Orchestrator: core.NewOrchestrator(timeAddition, timeSubtraction, timeMultiplication, timeDivision),
	}

	engine.POST("/api/v1/calculate", calculateHandler(server.Orchestrator))
	engine.GET("/api/v1/expressions", listExpressionsHandler(server.Orchestrator))
	engine.GET("/api/v1/expressions/:id", getExpressionHandler(server.Orchestrator))
	engine.GET("/internal/task", getTaskHandler(server.Orchestrator))
	engine.POST("/internal/task", submitTaskResultHandler(server.Orchestrator))

	return server
}

func (s *Server) Run(port string) error {
	return s.Engine.Run(port)
}

func resolveTask(o *core.Orchestrator, task *types.Task) types.TaskResponse {
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

	return types.TaskResponse{
		ID:            task.ID,
		Operation:     task.Operation,
		Arg1:          resolveArg(task.Arg1),
		Arg2:          resolveArg(task.Arg2),
		OperationTime: int(o.OperationTimes[task.Operation]),
	}
}

func calculateHandler(o *core.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Expression string `json:"expression" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request body"})
			logger.Error("invalid request body", zap.Error(err))
			return
		}

		err := core.ValidateExpression(req.Expression)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			logger.Error("invalid expression", zap.Error(err))
			return
		}

		exprID, err := o.AddExpression(req.Expression)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to process expression"})
			logger.Error("failed to process expression", zap.Error(err))
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": exprID})
	}
}

func listExpressionsHandler(o *core.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		expressions, err := o.GetAllExpressions()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get all expressions"})
			return
		}

		response := make([]types.ExpressionResponse, 0, len(expressions))

		for _, expr := range expressions {
			response = append(response, types.ExpressionResponse{
				ID:     expr.ID,
				Status: expr.Status,
				Result: expr.Result,
			})
		}

		c.JSON(http.StatusOK, gin.H{"expressions": response})
	}
}

func getExpressionHandler(o *core.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		exprID := c.Param("id")
		expr, exists, err := o.GetExpression(exprID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get expression"})
			return
		}
		if !exists {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "expression not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"expression": types.ExpressionResponse{
				ID:     expr.ID,
				Status: expr.Status,
				Result: expr.Result,
			},
		})
	}
}

func getTaskHandler(o *core.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		task, err := o.GetNextTask()
		if err != nil {
			if errors.Is(err, errs.ErrNoTasksAvailable) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "no tasks available"})
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			}
			return
		}

		resolvedTask := resolveTask(o, task)
		c.JSON(http.StatusOK, gin.H{"task": resolvedTask})
	}
}

func submitTaskResultHandler(o *core.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ID     string  `json:"id" binding:"required"`
			Result float64 `json:"result" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request body"})
			return
		}

		if err := o.ProcessTaskResult(req.ID, req.Result); err != nil {
			if errors.Is(err, errs.ErrTaskNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "task not found"})
			} else if errors.Is(err, errs.ErrInvalidTaskResult) {
				c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid task result"})
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			}
			return
		}

		c.Status(http.StatusOK)
	}
}
