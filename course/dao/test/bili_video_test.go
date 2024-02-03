package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/course/dao"
	"github.com/Orisun/radic/v2/util"
)

var csvFile = util.RootPath + "data/bili_video.csv"

func TestDumpDataFromFile2DB1(t *testing.T) {
	dao.DumpDataFromFile2DB1(csvFile) //DumpDataFromFile2DB1 use time 117240 ms
	/*
		select count(*) from bili_video;
		delete from bili_video;
	*/
}

func TestDumpDataFromFile2DB2(t *testing.T) {
	dao.DumpDataFromFile2DB2(csvFile) //DumpDataFromFile2DB2 use time 7955 ms
	/*
		select count(*) from bili_video;
		delete from bili_video;
	*/
}

func TestDumpDataFromFile2DB3(t *testing.T) {
	dao.DumpDataFromFile2DB3(csvFile) //DumpDataFromFile2DB3 use time 3367 ms
	/*
		select count(*) from bili_video;
		delete from bili_video;
	*/
}

func testReadAllTable(f func(ch chan<- dao.BiliVideo)) {
	ch := make(chan dao.BiliVideo, 100)
	go f(ch)
	idMap := make(map[string]struct{}, 40000)
	for {
		video, ok := <-ch
		if !ok {
			break
		}
		idMap[video.Id] = struct{}{}
	}
	fmt.Println(len(idMap))
}

func TestReadAllTable1(t *testing.T) {
	testReadAllTable(dao.ReadAllTable1) //ReadAllTable1 use time 173 ms
}

func TestReadAllTable2(t *testing.T) {
	testReadAllTable(dao.ReadAllTable2) //ReadAllTable2 use time 2654 ms
}

func TestReadAllTable3(t *testing.T) {
	testReadAllTable(dao.ReadAllTable3) //ReadAllTable3 use time 262 ms
}

// go test -v ./course/dao/test -run=^TestDumpDataFromFile2DB1$ -count=1
// go test -v ./course/dao/test -run=^TestDumpDataFromFile2DB2$ -count=1
// go test -v ./course/dao/test -run=^TestDumpDataFromFile2DB3$ -count=1
// go test -v ./course/dao/test -run=^TestReadAllTable1$ -count=1 -timeout=30m
// go test -v ./course/dao/test -run=^TestReadAllTable2$ -count=1 -timeout=30m
// go test -v ./course/dao/test -run=^TestReadAllTable3$ -count=1 -timeout=30m
