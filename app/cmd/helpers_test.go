package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateArgsValid(t *testing.T) {
	args := []string{"myqueue"}
	err := validateArgs(args, "queue name")
	require.NoError(t, err)
	assert.Equal(t, "myqueue", args[0])
}

func TestValidateArgsEmpty(t *testing.T) {
	args := []string{""}
	err := validateArgs(args, "queue name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue name cannot be empty")
}

func TestValidateArgsWhitespace(t *testing.T) {
	args := []string{"   "}
	err := validateArgs(args, "queue name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue name cannot be empty")
}

func TestValidateArgsTrimming(t *testing.T) {
	args := []string{"  hello  "}
	err := validateArgs(args, "name")
	require.NoError(t, err)
	assert.Equal(t, "hello", args[0])
}

func TestValidateArgsMultiple(t *testing.T) {
	args := []string{"vault1", "mykey", "myvalue"}
	err := validateArgs(args, "vault name", "key", "value")
	require.NoError(t, err)
	assert.Equal(t, "vault1", args[0])
	assert.Equal(t, "mykey", args[1])
	assert.Equal(t, "myvalue", args[2])
}

func TestValidateArgsMultipleSecondEmpty(t *testing.T) {
	args := []string{"vault1", "", "myvalue"}
	err := validateArgs(args, "vault name", "key", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")
}

func TestValidateArgsFewerArgsThanNames(t *testing.T) {
	args := []string{"onlyone"}
	err := validateArgs(args, "first", "second", "third")
	// Should not error — only checks args that exist
	require.NoError(t, err)
}

func TestValidateArgsNoNames(t *testing.T) {
	args := []string{"anything"}
	err := validateArgs(args)
	require.NoError(t, err)
}
