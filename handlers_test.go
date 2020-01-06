package g8_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	adapter "github.com/gaw508/lambda-proxy-http-adapter"
	"github.com/steinfletcher/apitest"

	"github.com/JSainsburyPLC/g8"
)

type body struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (b body) Validate() error {
	if b.Status == "" {
		return errors.New("status empty")
	}
	return nil
}

func TestBind(t *testing.T) {
	h := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		var b body
		err := g8.Bind(r.Body, &b)
		if err != nil {
			t.Fatalf("expected err to be nil. %s", err)
		}

		if b.Name != "Mei" {
			t.Fatal("expected body to contain 'name==Mei'")
		}

		if b.Status != "ACTIVE" {
			t.Fatal("expected body to contain 'status==ACTIVE'")
		}

		return g8.OK(nil)
	}

	apitest.New().
		Handler(adapter.GetHttpHandler(h, "/", nil)).
		Get("/").
		JSON(`{
			"name": "Mei",
			"status": "ACTIVE"
		}`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestBind_Validate(t *testing.T) {
	h := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		var b body
		err := g8.Bind(r.Body, &b)
		if err == nil {
			t.Fatal("expected err from validation")
		}

		if err.Error() != "status empty" {
			t.Fatalf("unexpected validation message: %s", err.Error())
		}

		return g8.OK(nil)
	}

	apitest.New().
		Handler(adapter.GetHttpHandler(h, "/", nil)).
		Get("/").
		JSON(`{
			"name": "Mei"
		}`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestBind_HandlesInvalidRequestJSON(t *testing.T) {
	h := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		var b body
		err := g8.Bind(r.Body, &b)
		if err == nil {
			t.Fatal("expected err from validation")
		}

		if !errors.Is(err, g8.ErrInvalidBody) {
			t.Fatal("expected ErrInvalidBody from validation")
		}

		return g8.JSON(http.StatusBadRequest, nil)
	}

	apitest.New().
		Handler(adapter.GetHttpHandler(h, "/", nil)).
		Get("/").
		JSON(`not json`).
		Expect(t).
		Status(http.StatusBadRequest).
		End()
}

func TestBind_HandlesInvalidResponseJSON(t *testing.T) {
	h := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		forceErrorValue := make(chan int)
		return g8.OK(forceErrorValue)
	}

	apitest.New().
		Handler(adapter.GetHttpHandler(h, "/", nil)).
		Get("/").
		Expect(t).
		Status(http.StatusInternalServerError).
		End()
}

func TestErrorResponse_InternalServer(t *testing.T) {
	h := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return g8.Error(errors.New("error"))
	}

	apitest.New().
		Handler(adapter.GetHttpHandler(h, "/", nil)).
		Get("/").
		Expect(t).
		Status(http.StatusInternalServerError).
		Body(`{"code":"INTERNAL_SERVER_ERROR", "detail":"Internal server error"}`).
		End()
}

func TestErrorResponse_CustomError(t *testing.T) {
	h := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return g8.Error(g8.Err{
			Status: http.StatusBadRequest,
			Code:   "INVALID_QUERY_PARAM",
			Detail: "Invalid query param",
		})
	}

	apitest.New().
		Handler(adapter.GetHttpHandler(h, "/", nil)).
		Get("/").
		Expect(t).
		Status(http.StatusBadRequest).
		Body(`{"code":"INVALID_QUERY_PARAM", "detail":"Invalid query param"}`).
		End()
}

func TestError_Error(t *testing.T) {
	err := g8.Err{
		Status: http.StatusBadRequest,
		Code:   "INVALID_QUERY_PARAM",
		Detail: "Invalid query param",
	}

	if err.Error() != "Code: INVALID_QUERY_PARAM; Status: 400; Detail: Invalid query param" {
		t.Fatalf("unexpected error: '%s'", err.Error())
	}
}
