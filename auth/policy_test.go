package auth

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasMethodsEmpty(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("principal-ID", "aws-account-id")

	// When:
	hasAllowMethods := resp.HasAllowingMethod()

	// Then:
	assert.False(t, hasAllowMethods)
}

func TestHasMethodsNonEmptyButContainsAllDenies(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("principal-ID", "aws-account-id")

	// and
	resp.DenyAllMethods()

	// When:
	hasAllowMethods := resp.HasAllowingMethod()

	// Then:
	assert.False(t, hasAllowMethods)
}

func TestHasMethodsAllowsAllMethods(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("principal-ID", "aws-account-id")

	// and
	resp.AllowAllMethods()

	// When:
	hasAllowMethods := resp.HasAllowingMethod()

	// Then:
	assert.True(t, hasAllowMethods)
}

func TestHasMethodsHasMixedAllowAndDenyMethods(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("principal-ID", "aws-account-id")

	// and
	resp.AllowMethod(Post, "/pets/*")
	resp.AllowMethod(Delete, "/cars/*")
	resp.AllowMethod(Get, "/users/*") // <-- !!!
	resp.AllowMethod(Post, "/picture/update")
	resp.AllowMethod(Post, "/picture/assign")
	resp.AllowMethod(Put, "/users/new")

	// When:
	hasAllowMethods := resp.HasAllowingMethod()

	// Then:
	assert.True(t, hasAllowMethods)
}
