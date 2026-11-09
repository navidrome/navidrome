package criteria

import (
	"encoding/json"
	"testing"
)

func TestMarshalEmptyNestedGroup(t *testing.T) {
	input := All{All{}}
	b, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"all":[{"all":[]}]}`
	if string(b) != expected {
		t.Fatalf("got %s, want %s", b, expected)
	}
}
