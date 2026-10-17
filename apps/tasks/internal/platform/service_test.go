package platform

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTrustGatewayRejectsDirectCalls(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/x", TrustGateway("secret"), func(c *gin.Context) {
		if Owner(c) != "u" {
			c.Status(500)
			return
		}
		c.Status(204)
	})
	for _, tc := range []struct {
		token, user string
		status      int
	}{{"", "", 401}, {"wrong", "u", 401}, {"secret", "", 401}, {"secret", "u", 204}} {
		req := httptest.NewRequest("GET", "/x", nil)
		req.Header.Set(HeaderToken, tc.token)
		req.Header.Set(HeaderUserID, tc.user)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Errorf("%+v: %d", tc, w.Code)
		}
	}
}
