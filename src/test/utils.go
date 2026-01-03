package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func CheckError(t *testing.T, result any) {
	_, ok := result.(error)
	if !ok {
		t.Errorf("expected error type, got %T", result)
	}

}
func CheckErrorString(t *testing.T, result any, expected error) {
	err, ok := result.(error)
	if ok {
		assert.EqualError(t, err, expected.Error())
	} else {
		t.Errorf("expected error type, got %T", result)
	}
}
func CheckErrorIs(t *testing.T, result any, expected error) {
	err, ok := result.(error)
	if ok {
		assert.ErrorIs(t, err, expected)
	} else {
		t.Errorf("expected error type, got %T", result)
	}
}
