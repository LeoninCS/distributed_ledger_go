package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleAuditEntry(c *gin.Context) {
	indexStr := c.Param("index")
	if indexStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "index required"})
		return
	}
	idx, err := strconv.ParseUint(indexStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid index: %v", err)})
		return
	}
	entry, err := s.auditSvc.GetEntry(idx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}
