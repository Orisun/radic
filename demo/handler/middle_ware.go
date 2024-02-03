package handler

import (
	"net/url"

	"github.com/gin-gonic/gin"
)

func GetUserInfo(ctx *gin.Context) {
	userName, err := url.QueryUnescape(ctx.Request.Header.Get("UserName")) //从request header里获得UserName
	if err == nil {
		ctx.Set("user_name", userName) //把UserName放到gin.Context里
	}
}
