package vtag

import (
	"fmt"
	"testing"
)

type Base struct {
	HelloWorld string `vtag:",list,detail"`
}

type Ext struct {
	DD string `vtag:",list"`
}

type Student struct {
	*Base
	Ext  Ext    `vtag:",list"`
	Name string `vtag:"name,list,detail"`
	Age  string `vtag:"age,list,detail"`
}

func TestMapWithTag(t *testing.T) {

	InitEncoder(UnderScoreCaseEncodeFunc)

	s, err := SliceWithTag(Student{}, "", "list")
	if err != nil {
		t.Error(err.Error())
		return
	}

	fmt.Println(s)

	s, err = SliceWithTag(Student{}, "", "detail")
	if err != nil {
		t.Error(err.Error())
		return
	}
	fmt.Println(s)
}
