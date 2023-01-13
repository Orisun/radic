// Date         : 2019-02-18 17:57
// Author       : zhangzhaoyang
// Description  :

package test

import (
	"fmt"
	"radic/types"
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestSerialize(t *testing.T) {
	request := types.SearchRequest{}
	request.Must = []*types.Keyword{&types.Keyword{Field: "CI", Word: "1"}}
	requestB, _ := proto.Marshal(&request)
	var request2 types.SearchRequest
	proto.Unmarshal(requestB, &request2)
	fmt.Println(request2.Must[0].Field)
	fmt.Println(request2.Must[0].Word)
}
