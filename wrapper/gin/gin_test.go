package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
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
	ctx.Request.Header.Add("Origin", "http://example.com/")
	ctx.Request.Header.Add("Access-Control-Request-Method", "POST")
	ctx.Status(http.StatusAccepted)
	res.Code = http.StatusAccepted

	handler := corsWrapper{Cors: cors.New(Options{
		// Intentionally left blank.
	})}.build()

	handler(ctx)

	if !ctx.IsAborted() {
		t.Error("Should abort on preflight requests")
	}
	if res.Code != http.StatusOK {
		t.Error("Should abort with 200 OK status")
	}
}

func TestCorsWrapper_buildNotAbortsWhenPassthrough(t *testing.T) {
	res := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(res)
	ctx.Request, _ = http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	ctx.Request.Header.Add("Origin", "http://example.com/")
	ctx.Request.Header.Add("Access-Control-Request-Method", "POST")

	handler := corsWrapper{cors.New(Options{
		OptionsPassthrough: true,
	}), true}.build()

	handler(ctx)

	if ctx.IsAborted() {
		t.Error("Should not abort when OPTIONS passthrough enabled")
	}
}
