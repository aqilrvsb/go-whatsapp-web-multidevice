package handlers

import (
	"net/http"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/usecase"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FlowUpdateHandler handles sequence flow updates
type FlowUpdateHandler struct {
	updater *usecase.SequenceFlowUpdater
}

// NewFlowUpdateHandler creates a new flow update handler
func NewFlowUpdateHandler() *FlowUpdateHandler {
	return &FlowUpdateHandler{
		updater: usecase.NewSequenceFlowUpdater(),
	}
}

// FlowUpdate processes flow update for a sequence
func (h *FlowUpdateHandler) FlowUpdate(c *gin.Context) {
	sequenceID := c.Param("sequenceId")
	if sequenceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Sequence ID is required",
		})
		return
	}

	logrus.Infof("Flow update requested for sequence: %s", sequenceID)

	// Execute flow update
	leadsUpdated, messagesCreated, err := h.updater.FlowUpdate(sequenceID)
	if err != nil {
		logrus.Errorf("Flow update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"leads_updated":     leadsUpdated,
		"messages_created":  messagesCreated,
		"message":          "Flow update completed successfully",
	})
}

// GetFlowStatus gets the flow status for a sequence
func (h *FlowUpdateHandler) GetFlowStatus(c *gin.Context) {
	sequenceID := c.Param("sequenceId")
	if sequenceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Sequence ID is required",
		})
		return
	}

	status, err := h.updater.GetSequenceFlowStatus(sequenceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}