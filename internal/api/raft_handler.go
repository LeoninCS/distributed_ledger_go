package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type raftJoinRequest struct {
	NodeID      string `json:"node_id"`
	RaftAddress string `json:"raft_address"`
}

type raftRemoveRequest struct {
	NodeID string `json:"node_id"`
}

func (s *Server) handleRaftJoin(c *gin.Context) {
	if s.joinFunc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "join unavailable"})
		return
	}
	var req raftJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.NodeID == "" || req.RaftAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id and raft_address required"})
		return
	}
	leader, err := s.joinFunc(req.NodeID, req.RaftAddress)
	if err != nil {
		if leader != "" {
			c.Header("X-Raft-Leader", leader)
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleRaftRemove(c *gin.Context) {
	if s.removeFunc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "remove unavailable"})
		return
	}
	var req raftRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.NodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id required"})
		return
	}
	leader, err := s.removeFunc(req.NodeID)
	if err != nil {
		if leader != "" {
			c.Header("X-Raft-Leader", leader)
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleRaftStatus(c *gin.Context) {
	if s.statusFunc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "status unavailable"})
		return
	}
	c.JSON(http.StatusOK, s.statusFunc())
}
