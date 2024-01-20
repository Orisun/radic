package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Orisun/radic/v2/demo/handler"
	"github.com/gin-gonic/gin"
)

var (
	rebuildIndex = flag.Bool("index", false, "rebuild index")
)

func listenSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	handler.CloseIndex() //接收到kill信号时关闭索引
	os.Exit(0)
}

func main() {
	flag.Parse()
	go listenSignal()
	if *rebuildIndex {
		handler.BuildIndex() //重建索引
	} else {
		handler.LoadIndex() //加载建好的索引
	}

	engine := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	engine.Static("js", "demo/views/js")
	engine.Static("css", "demo/views/css")
	engine.Static("img", "demo/views/img")
	engine.StaticFile("/favicon.ico", "img/dqq.png")                            //在url中访问文件/favicon.ico，相当于访问文件系统中的views/img/dqq.png文件
	engine.LoadHTMLFiles("demo/views/search.html", "demo/views/up_search.html") //使用这些.html文件时就不需要加路径了

	engine.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "search.html", []string{"资讯", "社会", "热点", "生活", "知识", "环球", "游戏", "综合", "日常", "影视", "科技", "编程"})
	})
	engine.GET("/up", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "up_search.html", []string{"资讯", "社会", "热点", "生活", "知识", "环球", "游戏", "综合", "日常", "影视", "科技", "编程"})
	})
	// engine.POST("/search", handler.Search)
	engine.POST("/search", handler.SearchAll)
	engine.POST("/up_search", handler.SearchByAuthor)
	engine.Run("127.0.0.1:5678")
}

// go run ./demo/main -index=false
