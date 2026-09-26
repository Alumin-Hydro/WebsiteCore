package httpx

import (
	"github.com/BZYA-Community/WebsiteCore/internal/model/joint"
	"github.com/BZYA-Community/WebsiteCore/pkg/xerror"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Render(c *gin.Context, data any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, &joint.JsonResp{
			Code: 0,
			Msg:  "success",
			Data: data,
		})
	} else {
		statusCode, code := xerror.HttpStatusCode(err)
		c.JSON(statusCode, &joint.JsonResp{
			Code: code,
			Msg:  err.Error(),
		})
	}
}
