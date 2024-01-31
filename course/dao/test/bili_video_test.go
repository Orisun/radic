package test

import (
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

// go test -v ./course/dao/test -run=^TestDumpDataFromFile2DB1$ -count=1
// go test -v ./course/dao/test -run=^TestDumpDataFromFile2DB2$ -count=1
// go test -v ./course/dao/test -run=^TestDumpDataFromFile2DB3$ -count=1
