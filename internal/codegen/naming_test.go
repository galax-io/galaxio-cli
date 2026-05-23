package codegen

import (
	"reflect"
	"testing"
)

func TestLowerCamel(t *testing.T) {
	t.Parallel()

	if got := LowerCamel("Pet Admin API"); got != "petAdminApi" {
		t.Fatalf("expected petAdminApi, got %q", got)
	}
}

func TestDeriveNameWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "Orders API", want: "ordersapi"},
		{input: "orders-api", want: "ordersapi"},
		{input: "", want: "generated"},
	}

	for _, tt := range tests {
		if got := DeriveNameWord(tt.input); got != tt.want {
			t.Fatalf("DeriveNameWord(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSplitWords(t *testing.T) {
	t.Parallel()

	got := SplitWords("PetAdmin/{petId}/create-order")
	want := []string{"Pet", "Admin", "pet", "Id", "create", "order"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitWords() = %#v, want %#v", got, want)
	}
}
