package test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	demo "github.com/Orisun/radic/v2/example"
	"github.com/bytedance/sonic"
)

func TestSearch(t *testing.T) {
	client := http.Client{
		Timeout: 100 * time.Millisecond,
	}
	request := demo.SearchRequest{
		Keywords: []string{"go", "gin"},
		Classes:  []string{"科技", "编程"},
		ViewFrom: 1000, //播放量大于1000
		ViewTo:   0,    //播放量不设上限
	}
	bs, _ := sonic.Marshal(request)
	resp, err := client.Post("http://127.0.0.1:5678/search", "application/json", bytes.NewReader(bs))
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		var datas []demo.BiliVideo
		sonic.Unmarshal(content, &datas)
		for _, data := range datas {
			fmt.Printf("%s %d %s %s\n", data.Id, data.View, data.Title, strings.Join(data.Keywords, "|"))
		}
	} else {
		fmt.Println(resp.Status)
		t.Fail()
	}
}

// go test -v ./demo/test -count=1 -run=^TestSearch$
