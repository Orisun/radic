package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/util"
)

func TestGetLocalIP(t *testing.T) {
	fmt.Println(util.GetLocalIP())
}

// go test -v ./util/test -run=^TestGetLocalIP$ -count=1
