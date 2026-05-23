package parser

import (
	"context"
	"strings"
	"testing"
)

func TestPostmanParserParseBuildsSpecFromCollection(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "sample-collection.json")

	spec, err := NewPostmanParser().Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if spec.Title != "Orders API Collection" {
		t.Fatalf("expected title %q, got %q", "Orders API Collection", spec.Title)
	}
	if spec.Version != "2.1.0" {
		t.Fatalf("expected version %q, got %q", "2.1.0", spec.Version)
	}
	if len(spec.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(spec.Groups))
	}

	orders := findGroup(t, spec, "orders")
	if len(orders.Requests) != 2 {
		t.Fatalf("expected 2 requests in orders group, got %d", len(orders.Requests))
	}

	createOrder := findRequest(t, orders, "createOrder")
	if createOrder.Method != "POST" {
		t.Fatalf("expected method %q, got %q", "POST", createOrder.Method)
	}
	if createOrder.Path != "/api/v1/orders" {
		t.Fatalf("expected path %q, got %q", "/api/v1/orders", createOrder.Path)
	}
	if createOrder.Body == nil {
		t.Fatal("expected createOrder body")
	}
	if !strings.Contains(createOrder.Body.Raw, `"sku": "${sku}"`) {
		t.Fatalf("expected variable mapping in raw body, got %q", createOrder.Body.Raw)
	}
	if len(createOrder.Comments) == 0 || !strings.Contains(createOrder.Comments[0], "TODO prerequest script:") {
		t.Fatalf("expected pre-request TODO comments, got %#v", createOrder.Comments)
	}

	getOrder := findRequest(t, orders, "getOrder")
	if getOrder.Path != "/api/v1/orders/${orderId}" {
		t.Fatalf("expected path placeholder mapping, got %q", getOrder.Path)
	}
	if len(getOrder.Params) != 1 || getOrder.Params[0].Name != "expand" {
		t.Fatalf("expected query param mapping, got %#v", getOrder.Params)
	}

	health := findGroup(t, spec, "default")
	if len(health.Requests) != 1 {
		t.Fatalf("expected 1 request in default group, got %d", len(health.Requests))
	}
	if health.Requests[0].Name != "healthCheck" {
		t.Fatalf("expected top-level request name %q, got %q", "healthCheck", health.Requests[0].Name)
	}

	if len(spec.AuthSchemes) != 2 {
		t.Fatalf("expected 2 auth scheme hints, got %d", len(spec.AuthSchemes))
	}
	bearer := findAuthScheme(t, spec, "collectionAuth")
	if bearer.Type != "bearer" {
		t.Fatalf("expected collection bearer auth, got %#v", bearer)
	}
	apiKey := findAuthScheme(t, spec, "ordersAuth")
	if apiKey.Type != "apikey" || apiKey.Name != "X-API-Key" || apiKey.Location != "header" {
		t.Fatalf("expected folder api key auth, got %#v", apiKey)
	}
}

func TestPostmanParserRejectsInvalidDocument(t *testing.T) {
	t.Parallel()

	_, err := NewPostmanParser().Parse(context.Background(), []byte(`{"info":{}}`))
	if err == nil {
		t.Fatal("expected invalid Postman document to return an error")
	}
}

func TestPostmanParserParseSupportsZitadelCollectionFixture(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "zitadel.postman_collection.json")

	spec, err := NewPostmanParser().Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if spec.Title != "ZITADEL with Postman" {
		t.Fatalf("expected title %q, got %q", "ZITADEL with Postman", spec.Title)
	}
	if spec.Version != "2.1.0" {
		t.Fatalf("expected version %q, got %q", "2.1.0", spec.Version)
	}

	defaultGroup := findGroup(t, spec, "default")
	if len(defaultGroup.Requests) != 7 {
		t.Fatalf("expected 7 top-level requests, got %d", len(defaultGroup.Requests))
	}

	addProject := findRequest(t, defaultGroup, "addZitadelProject")
	if addProject.Path != "${yourZitadelDomain}/management/v1/projects" {
		t.Fatalf("expected add project path, got %q", addProject.Path)
	}
	if addProject.Body == nil || !strings.Contains(addProject.Body.Raw, `"name": "MyPostmanProject"`) {
		t.Fatalf("expected raw body for add project, got %#v", addProject.Body)
	}

	loginUserinfo := findRequest(t, defaultGroup, "loginUserAndCallUserinfoEndpoint")
	if loginUserinfo.Path != "${yourZitadelDomain}/oidc/v1/userinfo" && loginUserinfo.Path != "/oidc/v1/userinfo" {
		t.Fatalf("unexpected userinfo path %q", loginUserinfo.Path)
	}

	if len(spec.AuthSchemes) == 0 {
		t.Fatal("expected auth schemes from per-request auth blocks")
	}
}
