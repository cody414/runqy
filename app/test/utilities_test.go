package test

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClamp(t *testing.T) {

	load := 5
	value := 6 + load/10
	fmt.Println(value)
	min := 3
	max := 8
	result := int(math.Max(float64(min), math.Min(float64(value), float64(max))))
	assert.Equal(t, 6, result)
}
