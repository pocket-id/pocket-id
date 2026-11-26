package ldapsync

import (
	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/actor"
)

type handler struct {
	actors *actor.Service
}

func (h *handler) syncLdap(c *gin.Context) error {
	// The sync runs in the actor's turn like the scheduled one, so on every replica a sync finishes before the next one reads the directory
	// A sync that read the directory earlier would otherwise write its older view over the newer result
	_, err := h.actors.Invoke(c.Request.Context(), SyncActorType, actor.SingletonActorID, methodSync, nil)
	return err
}
