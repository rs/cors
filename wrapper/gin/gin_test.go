package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func TestAllowAllNotNil(t *testing.T) {
	handler := AllowAll()
	if handler == nil {
		t.Error("Should not return nil Gin handler")
	}
}

func TestDefaultNotNil(t *testing.T) {
	handler := Default()
	if handler == nil {
		t.Error("Should not return nil Gin handler")
	}
}

func TestNewNotNil(t *testing.T) {
	handler := New(Options{})
	if handler == nil {
		t.Error("Should not return nil Gin handler")
	}
}

func TestCorsWrapper_buildAbortsWhenPreflight(t *testing.T) {
	res := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(res)
	ctx.Request, _ = http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	ctx.Request.Header.Add("Origin", "http://example.org")
	ctx.Request.Header.Add("Access-Control-Request-Method", "POST")
	ctx.Status(http.StatusAccepted)
	res.Code = http.StatusAccepted

	handler := New(Options{ /* Intentionally left blank. */ })

	handler(ctx)

	if !ctx.IsAborted() {
		t.Error("Should abort on preflight requests")
	}
	if res.Code != http.StatusOK {
		t.Error("Should abort with 200 OK status")
	}
}
