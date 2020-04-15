package auth

import (
	"github.com/aws/aws-lambda-go/events"
	"strings"
)

// AuthorizerResponse struct is used to build proper MethodARN policy
type AuthorizerResponse struct {
	events.APIGatewayCustomAuthorizerResponse

	// The region where the API is deployed. By default this is set to '*'
	Region string

	// The AWS account id the policy will be generated for. This is used to create the method ARNs.
	AccountID string

	// The API Gateway API id. By default this is set to '*'
	APIID string

	// The name of the stage used in the policy. By default this is set to '*'
	Stage string
}

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

func NewAuthorizerResponse(accountID string) AuthorizerResponse {
	return AuthorizerResponse{
		APIGatewayCustomAuthorizerResponse: events.APIGatewayCustomAuthorizerResponse{
			PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
				Version: "2012-10-17",
			},
		},
		Region:    "*",
		AccountID: accountID,
		APIID:     "*",
		Stage:     "*",
	}
}

func (r *AuthorizerResponse) buildResourceARN(verb, resource string) string {
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

func (r *AuthorizerResponse) addMethod(effect Effect, verb, resource string) {

	s := events.IAMPolicyStatement{
		Effect:   effect.String(),
		Action:   []string{"execute-api:Invoke"},
		Resource: []string{r.buildResourceARN(verb, resource)},
	}

	r.PolicyDocument.Statement = append(r.PolicyDocument.Statement, s)
}

func (r *AuthorizerResponse) SetPrincipalID(principalID string) {
	r.PrincipalID = principalID
}

func (r *AuthorizerResponse) AllowAllMethods() {
	r.addMethod(Allow, All, "*")
}

func (r *AuthorizerResponse) DenyAllMethods() {
	r.addMethod(Deny, All, "*")
}

func (r *AuthorizerResponse) AllowMethod(verb, resource string) {
	r.addMethod(Allow, verb, resource)
}

func (r *AuthorizerResponse) DenyMethod(verb, resource string) {
	r.addMethod(Deny, verb, resource)
}

// HasAllowingMethod returns true if there is at least one "allow" method added to policy
func (r *AuthorizerResponse) HasAllowingMethod() bool {
	if len(r.PolicyDocument.Statement) == 0 {
		return false
	}

	strAllow := Allow.String()
	for _, m := range r.PolicyDocument.Statement {
		if m.Effect == strAllow {
			return true
		}
	}

	return false
}
