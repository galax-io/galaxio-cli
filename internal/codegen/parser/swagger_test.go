package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/codegen"
)

func TestSwaggerParserParseBuildsSpecFromSwagger2(t *testing.T) {
	fixture := readTestFixture(t, "petstore-swagger2.json")

	spec, err := NewSwaggerParser().Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if spec.Title != "Swagger Petstore" {
		t.Fatalf("expected title %q, got %q", "Swagger Petstore", spec.Title)
	}
	if spec.Version != "1.0.7" {
		t.Fatalf("expected version %q, got %q", "1.0.7", spec.Version)
	}
	if spec.BasePath != "/v2" {
		t.Fatalf("expected base path %q, got %q", "/v2", spec.BasePath)
	}

	if len(spec.Groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(spec.Groups))
	}

	pet := findGroup(t, spec, "pet")
	if len(pet.Requests) != 8 {
		t.Fatalf("expected 8 requests in pet group, got %d", len(pet.Requests))
	}

	addPet := findRequest(t, pet, "addPet")
	if addPet.Method != "POST" {
		t.Fatalf("expected addPet method %q, got %q", "POST", addPet.Method)
	}
	if addPet.Path != "/pet" {
		t.Fatalf("expected addPet path %q, got %q", "/pet", addPet.Path)
	}
	if addPet.Body == nil {
		t.Fatal("expected addPet to have a request body")
	}
	if addPet.Body.Type != "object" {
		t.Fatalf("expected body type %q, got %q", "object", addPet.Body.Type)
	}

	nameField := findField(t, addPet.Body, "name")
	if !nameField.Required {
		t.Fatal("expected name field to be required")
	}
	if nameField.Type != "string" {
		t.Fatalf("expected name field type %q, got %q", "string", nameField.Type)
	}

	photoURLsField := findField(t, addPet.Body, "photoUrls")
	if !photoURLsField.Required {
		t.Fatal("expected photoUrls field to be required")
	}
	if photoURLsField.Type != "array" {
		t.Fatalf("expected photoUrls field type %q, got %q", "array", photoURLsField.Type)
	}
	if photoURLsField.Items == nil || photoURLsField.Items.Type != "string" {
		t.Fatalf("expected photoUrls items type %q, got %#v", "string", photoURLsField.Items)
	}

	categoryField := findField(t, addPet.Body, "category")
	categoryNameField := findField(t, categoryField, "name")
	if categoryNameField.Type != "string" {
		t.Fatalf("expected nested category.name type %q, got %q", "string", categoryNameField.Type)
	}

	tagsField := findField(t, addPet.Body, "tags")
	if tagsField.Type != "array" {
		t.Fatalf("expected tags field type %q, got %q", "array", tagsField.Type)
	}
	if tagsField.Items == nil {
		t.Fatal("expected tags items schema to be resolved")
	}
	tagNameField := findField(t, tagsField.Items, "name")
	if tagNameField.Type != "string" {
		t.Fatalf("expected tag item name type %q, got %q", "string", tagNameField.Type)
	}

	if len(spec.AuthSchemes) != 2 {
		t.Fatalf("expected 2 auth schemes, got %d", len(spec.AuthSchemes))
	}

	apiKey := findAuthScheme(t, spec, "api_key")
	if apiKey.Type != "apikey" {
		t.Fatalf("expected api key auth type %q, got %q", "apikey", apiKey.Type)
	}
	if apiKey.Name != "api_key" {
		t.Fatalf("expected api key header name %q, got %q", "api_key", apiKey.Name)
	}
	if apiKey.Location != "header" {
		t.Fatalf("expected api key location %q, got %q", "header", apiKey.Location)
	}

	oauth := findAuthScheme(t, spec, "petstore_auth")
	if oauth.Type != "oauth2" {
		t.Fatalf("expected oauth auth type %q, got %q", "oauth2", oauth.Type)
	}
}

func TestSwaggerParserParseCapturesRequestShapesFromOfficialFixture(t *testing.T) {
	fixture := readTestFixture(t, "petstore-swagger2.json")

	spec, err := NewSwaggerParser().Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	pet := findGroup(t, spec, "pet")
	deletePet := findRequest(t, pet, "deletePet")
	if len(deletePet.Headers) != 1 {
		t.Fatalf("expected 1 header on deletePet, got %d", len(deletePet.Headers))
	}
	if deletePet.Headers[0].Name != "api_key" {
		t.Fatalf("expected header %q, got %q", "api_key", deletePet.Headers[0].Name)
	}
	if len(deletePet.Params) != 1 {
		t.Fatalf("expected 1 non-header param on deletePet, got %d", len(deletePet.Params))
	}
	if deletePet.Params[0].Name != "petId" {
		t.Fatalf("expected path param %q, got %q", "petId", deletePet.Params[0].Name)
	}

	store := findGroup(t, spec, "store")
	if len(store.Requests) != 4 {
		t.Fatalf("expected 4 requests in store group, got %d", len(store.Requests))
	}
	placeOrder := findRequest(t, store, "placeOrder")
	if placeOrder.Body == nil {
		t.Fatal("expected placeOrder to have a request body")
	}
	shipDateField := findField(t, placeOrder.Body, "shipDate")
	if shipDateField.Format != "date-time" {
		t.Fatalf("expected shipDate format %q, got %q", "date-time", shipDateField.Format)
	}

	inventory := findRequest(t, store, "getInventory")
	if len(inventory.Responses) != 1 {
		t.Fatalf("expected 1 response on getInventory, got %d", len(inventory.Responses))
	}

	user := findGroup(t, spec, "user")
	if len(user.Requests) != 8 {
		t.Fatalf("expected 8 requests in user group, got %d", len(user.Requests))
	}
	loginUser := findRequest(t, user, "loginUser")
	if len(loginUser.Params) != 2 {
		t.Fatalf("expected 2 query params on loginUser, got %d", len(loginUser.Params))
	}
	if loginUser.Params[0].Name != "username" || loginUser.Params[1].Name != "password" {
		t.Fatalf("expected loginUser params username/password, got %#v", loginUser.Params)
	}

	uploadFile := findRequest(t, pet, "uploadFile")
	if uploadFile.Path != "/pet/{petId}/uploadImage" {
		t.Fatalf("expected uploadFile path %q, got %q", "/pet/{petId}/uploadImage", uploadFile.Path)
	}
	if len(uploadFile.Params) != 3 {
		t.Fatalf("expected 3 params on uploadFile, got %d", len(uploadFile.Params))
	}

	createUsersWithListInput := findRequest(t, user, "createUsersWithListInput")
	if createUsersWithListInput.Body == nil {
		t.Fatal("expected createUsersWithListInput to have a body")
	}
	if createUsersWithListInput.Body.Type != "array" {
		t.Fatalf("expected createUsersWithListInput body type %q, got %q", "array", createUsersWithListInput.Body.Type)
	}
	if createUsersWithListInput.Body.Items == nil {
		t.Fatal("expected createUsersWithListInput array items to be resolved")
	}
	usernameField := findField(t, createUsersWithListInput.Body.Items, "username")
	if usernameField.Type != "string" {
		t.Fatalf("expected nested username type %q, got %q", "string", usernameField.Type)
	}
}

func TestSwaggerParserParseSupportsExpandedSwagger2Fixture(t *testing.T) {
	fixture := readTestFixture(t, "petstore-expanded.json")

	spec, err := NewSwaggerParser().Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if spec.Title != "Swagger Petstore" {
		t.Fatalf("expected title %q, got %q", "Swagger Petstore", spec.Title)
	}
	if spec.Version != "1.0.0" {
		t.Fatalf("expected version %q, got %q", "1.0.0", spec.Version)
	}
	if spec.BasePath != "/api" {
		t.Fatalf("expected base path %q, got %q", "/api", spec.BasePath)
	}
	if len(spec.AuthSchemes) != 0 {
		t.Fatalf("expected no auth schemes, got %d", len(spec.AuthSchemes))
	}

	pets := findGroup(t, spec, "pets")
	if len(pets.Requests) != 4 {
		t.Fatalf("expected 4 requests in pets group, got %d", len(pets.Requests))
	}

	addPet := findRequest(t, pets, "addPet")
	if addPet.Body == nil {
		t.Fatal("expected addPet body to be parsed")
	}
	nameField := findField(t, addPet.Body, "name")
	if !nameField.Required {
		t.Fatal("expected expanded fixture body field name to be required")
	}
	tagField := findField(t, addPet.Body, "tag")
	if tagField.Type != "string" {
		t.Fatalf("expected tag field type %q, got %q", "string", tagField.Type)
	}

	findPetByID := findRequest(t, pets, "findPetById")
	if findPetByID.Path != "/pets/{id}" {
		t.Fatalf("expected fallback path %q, got %q", "/pets/{id}", findPetByID.Path)
	}
	if len(findPetByID.Params) != 1 {
		t.Fatalf("expected 1 path param on findPetById, got %d", len(findPetByID.Params))
	}
	if findPetByID.Params[0].Name != "id" {
		t.Fatalf("expected path param %q, got %q", "id", findPetByID.Params[0].Name)
	}

	deletePet := findRequest(t, pets, "deletePet")
	if len(deletePet.Responses) != 2 {
		t.Fatalf("expected 2 responses on deletePet, got %d", len(deletePet.Responses))
	}
}

func readTestFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", path, err)
	}

	return data
}

func findGroup(t *testing.T, spec *codegen.Spec, name string) codegen.Group {
	t.Helper()

	for _, group := range spec.Groups {
		if group.Name == name {
			return group
		}
	}

	t.Fatalf("expected group %q to exist", name)
	return codegen.Group{}
}

func findRequest(t *testing.T, group codegen.Group, name string) codegen.Request {
	t.Helper()

	for _, request := range group.Requests {
		if request.Name == name {
			return request
		}
	}

	t.Fatalf("expected request %q to exist in group %q", name, group.Name)
	return codegen.Request{}
}

func findField(t *testing.T, schema *codegen.BodySchema, name string) *codegen.BodySchema {
	t.Helper()

	for i := range schema.Fields {
		if schema.Fields[i].Name == name {
			return &schema.Fields[i]
		}
	}

	t.Fatalf("expected field %q to exist", name)
	return nil
}

func findAuthScheme(t *testing.T, spec *codegen.Spec, key string) codegen.AuthScheme {
	t.Helper()

	for _, scheme := range spec.AuthSchemes {
		if scheme.Key == key {
			return scheme
		}
	}

	t.Fatalf("expected auth scheme %q to exist", key)
	return codegen.AuthScheme{}
}
