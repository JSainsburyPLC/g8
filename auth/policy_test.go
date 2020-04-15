package auth

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasMethodsEmpty(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("aws-account-id")

	// When:
	hasAllowMethods := resp.HasAllowingMethod()

	// Then:
	assert.False(t, hasAllowMethods)
}

func TestHasMethodsNonEmptyButContainsAllDenies(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("aws-account-id")

	// and
	resp.DenyAllMethods()

	// When:
	hasAllowMethods := resp.HasAllowingMethod()

	// Then:
	assert.False(t, hasAllowMethods)
}

func TestHasMethodsAllowsAllMethods(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("aws-account-id")

	// and
	resp.AllowAllMethods()

	// When:
	hasAllowMethods := resp.HasAllowingMethod()

	// Then:
	assert.True(t, hasAllowMethods)
}

func TestHasMethodsHasMixedAllowAndDenyMethods(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("aws-account-id")

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

func TestBuildResourceArn(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("aws-account-id")
	resp.SetPrincipalID("principal-id")

	// When
	resourceARN := resp.buildResourceARN(Post, "/pets/*")

	// Then:
	assert.Equal(t, "arn:aws:execute-api:*:aws-account-id:*/*/POST/pets/*", resourceARN)
}

func TestBuildResourceArnAllowAll(t *testing.T) {

	// Given:
	resp := NewAuthorizerResponse("aws-account-id")
	resp.SetPrincipalID("principal-id")

	// When
	resourceARN := resp.buildResourceARN(All, "*")

	// Then:
	assert.Equal(t, "arn:aws:execute-api:*:aws-account-id:*/*/*/*", resourceARN)
}
