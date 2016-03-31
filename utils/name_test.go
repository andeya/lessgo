package utils

import (
	"testing"
)

// snake string, XxYy to xx_yy
func TestSnakeString(t *testing.T) {
	t.Log(SnakeString("IndexHandle"))
}

// camel string, xx_yy to XxYy
func TestCamelString(t *testing.T) {
	t.Log(CamelString("index_handle"))
}
