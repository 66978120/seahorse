package errors

import (
	"fmt"
	"testing"
)

func A() error {
	return Errorf(nil, "a()")
}

func B() error {
	return Errorf(A(), "b()")
}
func TestErr(t *testing.T) {
	err := B()
	fmt.Println(err)
}
