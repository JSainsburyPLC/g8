package g8

import (
	"github.com/aws/aws-lambda-go/events"
	"strings"
)

const All = "*"

type Effect int

const (
	Allow Effect = iota
	Deny
)

func (e Effect) String() string {
	switch e {
	case Allow:
		return "Allow"
	case Deny:
		return "Deny"
	}
	return ""
}

type methodARN struct {

	// The region where the API is deployed. By default this is set to '*'
	Region string

	// The AWS account id the policy will be generated for. This is used to create the method ARNs.
	AccountID string

	// The API Gateway API id. By default this is set to '*'
	APIID string

	// The name of the stage used in the policy. By default this is set to '*'
	Stage string
}

func parseFromMethodARN(rawArn string) methodARN {

	tmp := strings.Split(rawArn, ":")
	apiGatewayArnTmp := strings.Split(tmp[5], "/")
	awsAccountID := tmp[4]

	return methodARN{
		AccountID: awsAccountID,
		Region:    tmp[3],
		APIID:     apiGatewayArnTmp[0],
		Stage:     apiGatewayArnTmp[1],
	}
}

func (r *methodARN) buildResourceARN(verb, resource string) string {
	var str strings.Builder

	str.WriteString("arn:aws:execute-api:")
	str.WriteString(r.Region)
	str.WriteString(":")
	str.WriteString(r.AccountID)
	str.WriteString(":")
	str.WriteString(r.APIID)
	str.WriteString("/")
	str.WriteString(r.Stage)
	str.WriteString("/")
	str.WriteString(verb)
	str.WriteString("/")
	str.WriteString(strings.TrimLeft(resource, "/"))

	return str.String()
}

func NewAuthorizerResponse() events.APIGatewayCustomAuthorizerResponse {
	return events.APIGatewayCustomAuthorizerResponse{
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
		},
	}
}

func (c *APIGatewayCustomAuthorizerContext) addMethod(effect Effect, verb, resource string) {
	s := events.IAMPolicyStatement{
		Effect:   effect.String(),
		Action:   []string{"execute-api:Invoke"},
		Resource: []string{c.methodArnParts.buildResourceARN(verb, resource)},
	}

	c.Response.PolicyDocument.Statement = append(c.Response.PolicyDocument.Statement, s)
}

func (c *APIGatewayCustomAuthorizerContext) SetPrincipalID(principalID string) {
	c.Response.PrincipalID = principalID
}

func (c *APIGatewayCustomAuthorizerContext) AllowAllMethods() {
	c.hasAtLeastOneAllowedMethod = true
	c.addMethod(Allow, All, "*")
}

func (c *APIGatewayCustomAuthorizerContext) DenyAllMethods() {
	c.addMethod(Deny, All, "*")
}

func (c *APIGatewayCustomAuthorizerContext) AllowMethod(verb, resource string) {
	c.hasAtLeastOneAllowedMethod = true
	c.addMethod(Allow, verb, resource)
}

func (c *APIGatewayCustomAuthorizerContext) DenyMethod(verb, resource string) {
	c.addMethod(Deny, verb, resource)
}
