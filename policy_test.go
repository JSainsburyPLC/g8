package g8

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestHasMethodsEmpty(t *testing.T) {

	// Given:
	c := APIGatewayCustomAuthorizerContext{}

	// When:
	hasAllowMethods := c.HasAllowingMethod()

	// Then:
	assert.False(t, hasAllowMethods)
}

func TestHasMethodsNonEmptyButContainsAllDenies(t *testing.T) {

	// Given:
	c := APIGatewayCustomAuthorizerContext{}

	// and
	c.DenyAllMethods()

	// When:
	hasAllowMethods := c.HasAllowingMethod()

	// Then:
	assert.False(t, hasAllowMethods)
}

func TestHasMethodsAllowsAllMethods(t *testing.T) {

	// Given:
	c := APIGatewayCustomAuthorizerContext{}

	// and
	c.AllowAllMethods()

	// When:
	hasAllowMethods := c.HasAllowingMethod()

	// Then:
	assert.True(t, hasAllowMethods)
}

func TestHasMethodsHasMixedAllowAndDenyMethods(t *testing.T) {

	// Given:
	c := APIGatewayCustomAuthorizerContext{}

	// and
	c.AllowMethod(http.MethodPost, "/pets/*")
	c.AllowMethod(http.MethodDelete, "/cars/*")
	c.AllowMethod(http.MethodGet, "/users/*") // <-- !!!
	c.AllowMethod(http.MethodPost, "/picture/update")
	c.AllowMethod(http.MethodPost, "/picture/assign")
	c.AllowMethod(http.MethodPut, "/users/new")

	// When:
	hasAllowMethods := c.HasAllowingMethod()

	// Then:
	assert.True(t, hasAllowMethods)
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
