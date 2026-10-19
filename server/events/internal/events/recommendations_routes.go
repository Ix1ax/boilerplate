package events

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

func recommendationPagination(c *gin.Context) (int, int, bool) {
	limit, e1 := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, e2 := strconv.Atoi(c.DefaultQuery("offset", "0"))
	return limit, offset, e1 == nil && e2 == nil && limit >= 1 && limit <= 50 && offset >= 0 && offset <= 1000
}
