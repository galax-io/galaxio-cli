package renderer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/codegen"
	"github.com/galax-io/galaxio-cli/internal/codegen/parser"
)

func TestRendererRenderMatchesGoldenFiles(t *testing.T) {
	t.Parallel()

	renderer := NewRenderer()
	files, err := renderer.Render(testSpec(), RenderOptions{Package: "org.galaxio.performance"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	wantFiles := map[string]string{
		"cases/AuthActions.scala":          "golden/cases/AuthActions.scala",
		"cases/PetAdminActions.scala":      "golden/cases/PetAdminActions.scala",
		"resources/bodies/createPet.json":  "golden/resources/bodies/createPet.json",
		"scenarios/PetAdminScenario.scala": "golden/scenarios/PetAdminScenario.scala",
	}

	if len(files) != len(wantFiles) {
		t.Fatalf("expected %d files, got %d", len(wantFiles), len(files))
	}

	got := make(map[string]string, len(files))
	for _, file := range files {
		got[file.Path] = string(file.Content)
	}

	for path, goldenPath := range wantFiles {
		payload, err := os.ReadFile(filepath.Join("testdata", goldenPath))
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", goldenPath, err)
		}

		if got[path] != string(payload) {
			t.Fatalf("unexpected output for %s\nwant:\n%s\ngot:\n%s", path, payload, got[path])
		}
	}
}

func TestRendererRenderFromSwaggerFixtureProducesScalaOutputs(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile(filepath.Join("..", "testdata", "petstore-swagger2.json"))
	if err != nil {
		t.Fatalf("ReadFile(fixture) error = %v", err)
	}

	spec, err := parser.NewSwaggerParser().Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	files, err := NewRenderer().Render(spec, RenderOptions{Package: "org.example.petstore"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	for _, wantPath := range []string{
		"cases/AuthActions.scala",
		"cases/PetActions.scala",
		"cases/StoreActions.scala",
		"cases/UserActions.scala",
		"scenarios/PetScenario.scala",
		"scenarios/StoreScenario.scala",
		"scenarios/UserScenario.scala",
		"resources/bodies/addPet.json",
		"resources/bodies/updatePet.json",
		"resources/bodies/placeOrder.json",
		"resources/bodies/updateUser.json",
		"resources/bodies/createUsersWithArrayInput.json",
		"resources/bodies/createUsersWithListInput.json",
		"resources/bodies/createUser.json",
	} {
		assertHasFileContaining(t, files, wantPath, "")
	}

	assertHasFileContaining(t, files, "cases/PetActions.scala", "package org.example.petstore.cases")
	assertHasFileContaining(t, files, "cases/PetActions.scala", `.body(ElFileBody("bodies/addPet.json")).asJson`)
	assertHasFileContaining(t, files, "scenarios/UserScenario.scala", "object UserScenario")
	assertHasFileContaining(t, files, "resources/bodies/placeOrder.json", `"shipDate": "${shipDate}"`)
	assertHasFileContaining(t, files, "resources/bodies/createUsersWithListInput.json", `"username": "${itemUsername}"`)
	assertHasFileContaining(t, files, "cases/AuthActions.scala", "Suggested schemes from the source spec")
}

func TestRendererRenderRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	renderer := NewRenderer()
	if _, err := renderer.Render(nil, RenderOptions{Package: "org.example"}); err == nil {
		t.Fatal("expected nil spec to fail")
	}
	if _, err := renderer.Render(&codegen.Spec{}, RenderOptions{}); err == nil {
		t.Fatal("expected empty package to fail")
	}
}

func assertHasFileContaining(t *testing.T, files []OutputFile, path string, want string) {
	t.Helper()

	for _, file := range files {
		if file.Path == path {
			if !strings.Contains(string(file.Content), want) {
				t.Fatalf("expected %s to contain %q, got:\n%s", path, want, file.Content)
			}
			return
		}
	}

	t.Fatalf("expected file %s to be rendered", path)
}

func testSpec() *codegen.Spec {
	return &codegen.Spec{
		BasePath: "/api/v1",
		AuthSchemes: []codegen.AuthScheme{
			{
				Key:      "bearerAuth",
				Type:     "bearer",
				Name:     "Authorization",
				Location: "header",
			},
		},
		Groups: []codegen.Group{
			{
				Name: "pet-admin",
				Requests: []codegen.Request{
					{
						Method: "POST",
						Path:   "/pets/{petId}",
						Name:   "createPet",
						Headers: []codegen.Header{
							{Name: "Authorization"},
						},
						Params: []codegen.Param{
							{Name: "petId", In: "path", Type: "string", Required: true},
							{Name: "trace-id", In: "query", Type: "string"},
						},
						Body: &codegen.BodySchema{
							Type: "object",
							Fields: []codegen.BodySchema{
								{Name: "name", Type: "string", Required: true},
								{
									Name: "category",
									Type: "object",
									Fields: []codegen.BodySchema{
										{Name: "id", Type: "integer"},
										{Name: "name", Type: "string"},
									},
								},
								{
									Name: "tags",
									Type: "array",
									Items: &codegen.BodySchema{
										Type: "object",
										Fields: []codegen.BodySchema{
											{Name: "id", Type: "integer"},
											{Name: "name", Type: "string"},
										},
									},
								},
							},
						},
						Responses: []codegen.Response{{StatusCode: "201"}},
					},
				},
			},
		},
	}
}
