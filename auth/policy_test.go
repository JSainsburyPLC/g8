package auth

import (
	"github.com/stretchr/testify/assert"
	"net/http"
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
	resp.AllowMethod(http.MethodPost, "/pets/*")
	resp.AllowMethod(http.MethodDelete, "/cars/*")
	resp.AllowMethod(http.MethodGet, "/users/*") // <-- !!!
	resp.AllowMethod(http.MethodPost, "/picture/update")
	resp.AllowMethod(http.MethodPost, "/picture/assign")
	resp.AllowMethod(http.MethodPut, "/users/new")

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
	resourceARN := resp.buildResourceARN(http.MethodPost, "/pets/*")

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
