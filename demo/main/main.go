package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Orisun/radic/v2/demo"
	"github.com/gin-gonic/gin"
)

var (
	rebuildIndex = flag.Bool("index", false, "rebuild index")
)

func listenSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	demo.CloseIndex() //接收到kill信号时关闭索引
}

func main() {
	flag.Parse()
	go listenSignal()
	if *rebuildIndex {
		demo.BuildIndex() //重建索引
	} else {
		demo.LoadIndex() //加载建好的索引
	}

	engine := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	engine.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "search.html", nil)
	})
	engine.POST("/search", demo.Search)
	engine.Run(":5678")
}

// go run ./demo/main -index=false
