package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitStatements(t *testing.T) {
	assert.Equal(t, []string{"CREATE TABLE a (id INT)"}, splitStatements("CREATE TABLE a (id INT);"))
	assert.Equal(t, []string{"A", "B", "C"}, splitStatements("A; B; C;"))
	assert.Equal(t, []string{"SELECT 1"}, splitStatements("  SELECT 1;  "))
	assert.Empty(t, splitStatements(""))
	assert.Empty(t, splitStatements("   ;   "))
	assert.Equal(t, []string{"A", "B"}, splitStatements("A;;B;"))
}
