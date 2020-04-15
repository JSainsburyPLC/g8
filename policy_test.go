package g8

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestHasMethodsEmpty(t *testing.T) {

	// Given:
	c := APIGatewayCustomAuthorizerContext{}

	// Then:
	assert.False(t, c.hasAtLeastOneAllowedMethod) // <-- default value is false
}

func TestHasMethodsNonEmptyButContainsAllDenies(t *testing.T) {

	// Given:
	c := APIGatewayCustomAuthorizerContext{}

	// When
	c.DenyAllMethods()

	// Then:
	assert.False(t, c.hasAtLeastOneAllowedMethod)
}

func TestHasMethodsAllowsAllMethods(t *testing.T) {

	// Given:
	c := APIGatewayCustomAuthorizerContext{}

	// When
	c.AllowAllMethods()

	// Then:
	assert.True(t, c.hasAtLeastOneAllowedMethod)
}

func TestHasMethodsHasMixedAllowAndDenyMethods(t *testing.T) {

	// Given:
	c := APIGatewayCustomAuthorizerContext{}

	// When
	c.DenyMethod(http.MethodPost, "/pets/*")
	c.DenyMethod(http.MethodDelete, "/cars/*")
	c.AllowMethod(http.MethodGet, "/users/*") // <-- !!!
	c.DenyMethod(http.MethodPost, "/picture/update")
	c.DenyMethod(http.MethodPost, "/picture/assign")
	c.DenyMethod(http.MethodPut, "/users/new")

	// Then:
	assert.True(t, c.hasAtLeastOneAllowedMethod)
}

func TestBuildResourceArn(t *testing.T) {

	// Given:
	m := methodARN{
		Region:    "eu-west-1",
		AccountID: "aws-account-id",
		APIID:     "*",
		Stage:     "*",
	}

	// When
	resourceARN := m.buildResourceARN(http.MethodPost, "/pets/*")

	// Then:
	assert.Equal(t, "arn:aws:execute-api:eu-west-1:aws-account-id:*/*/POST/pets/*", resourceARN)
}

func TestBuildResourceArnAllowAll(t *testing.T) {

	// Given:
	m := methodARN{
		Region:    "*",
		AccountID: "aws-account-id",
		APIID:     "*",
		Stage:     "*",
	}

	// When
	resourceARN := m.buildResourceARN(All, "*")

	// Then:
	assert.Equal(t, "arn:aws:execute-api:*:aws-account-id:*/*/*/*", resourceARN)
}

func TestParseMethodARN(t *testing.T) {

	// Given:
	strMethodARN := "arn:aws:execute-api:eu-west-1:123456789012:oy1e34abcd/main/GET/test-endpoint"

	// When:
	methodARN := parseFromMethodARN(strMethodARN)

	// Then:
	assert.Equal(t, "eu-west-1", methodARN.Region)
	assert.Equal(t, "123456789012", methodARN.AccountID)
	assert.Equal(t, "oy1e34abcd", methodARN.APIID)
	assert.Equal(t, "main", methodARN.Stage)
}
