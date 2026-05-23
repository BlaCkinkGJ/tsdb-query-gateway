package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/client"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

type QueryHandler struct {
	queryService service.QueryService
}

func NewQueryHandler(queryService service.QueryService) *QueryHandler {
	return &QueryHandler{
		queryService: queryService,
	}
}

func (h *QueryHandler) HandleQuery(c *gin.Context) {
	req := models.PromQueryRequest{
		Query:   c.Query("query"),
		Time:    c.Query("time"),
		Timeout: c.Query("timeout"),
	}

	if req.Query == "" {
		req.Query = c.PostForm("query")
		req.Time = c.PostForm("time")
		req.Timeout = c.PostForm("timeout")
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "errorType": "bad_data", "error": "missing query"})
		return
	}

	res, err := h.queryService.Query(c.Request.Context(), req)
	if err != nil {
		writeErrorResponse(c, "query", err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *QueryHandler) HandleQueryRange(c *gin.Context) {
	req := models.PromQueryRangeRequest{
		Query:   c.Query("query"),
		Start:   c.Query("start"),
		End:     c.Query("end"),
		Step:    c.Query("step"),
		Timeout: c.Query("timeout"),
	}

	if req.Query == "" {
		req.Query = c.PostForm("query")
		req.Start = c.PostForm("start")
		req.End = c.PostForm("end")
		req.Step = c.PostForm("step")
		req.Timeout = c.PostForm("timeout")
	}

	if req.Query == "" || req.Start == "" || req.End == "" || req.Step == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "errorType": "bad_data", "error": "missing required parameters"})
		return
	}

	res, err := h.queryService.QueryRange(c.Request.Context(), req)
	if err != nil {
		writeErrorResponse(c, "query_range", err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func writeErrorResponse(c *gin.Context, logPrefix string, err error) {
	log.Printf("%s error: %v", logPrefix, err)
	statusCode := http.StatusInternalServerError
	errorType := "server_error"
	msg := err.Error()
	var downstreamErr *client.DownstreamError
	if errors.As(err, &downstreamErr) {
		statusCode = downstreamErr.StatusCode
		errorType = downstreamErr.ErrorType
		msg = downstreamErr.Message
	}
	c.JSON(statusCode, gin.H{"status": "error", "errorType": errorType, "error": msg})
}

func (h *QueryHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/query", h.HandleQuery)
	router.POST("/query", h.HandleQuery)
	router.GET("/query_range", h.HandleQueryRange)
	router.POST("/query_range", h.HandleQueryRange)
}
