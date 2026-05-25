package management

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func (h *Handler) PostAPIKey(c *gin.Context) {
	var body struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}
	key := strings.TrimSpace(body.Key)
	if key == "" {
		c.JSON(400, gin.H{"error": "key is required"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Add if not exists
	found := false
	for _, k := range h.cfg.APIKeys {
		if k == key {
			found = true
			break
		}
	}
	if !found {
		h.cfg.APIKeys = append(h.cfg.APIKeys, key)
	}

	h.persistLocked(c)
}
