package resumption

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecoveryMatchesSuppliedExampleContracts(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/cli-wayfinding/examples/resumption-results.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples struct {
		Examples []struct {
			Case       string         `json:"case"`
			Projection map[string]any `json:"reportProjection"`
		} `json:"examples"`
	}
	if err := json.Unmarshal(data, &examples); err != nil {
		t.Fatal(err)
	}
	for _, example := range examples.Examples {
		t.Run(example.Case, func(t *testing.T) {
			request, retained := graphFixture(t)
			switch example.Case {
			case "absent":
			case "available":
				graphCheckpoint(t, request, retained, 1)
			case "conflicting":
				graphCheckpoint(t, request, retained, 1)
				graphCheckpoint(t, request, retained, 2)
			case "invalid snapshot":
				directory := graphCheckpoint(t, request, retained, 1)
				writeResumeFile(t, filepath.Join(directory, "context.json"), "invalid bytes")
			case "incomplete inventory":
				directory := graphCheckpoint(t, request, retained, 1)
				if err := os.Mkdir(filepath.Join(filepath.Dir(directory), graphID(2)), 0700); err != nil {
					t.Fatal(err)
				}
			default:
				t.Fatalf("unknown supplied example %q", example.Case)
			}
			result := resumeCheckpoint(t, request)
			encoded, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var actual map[string]any
			if err := json.Unmarshal(encoded, &actual); err != nil {
				t.Fatal(err)
			}
			assertProjectionShape(t, "report", actual, example.Projection)
			expectedRecovery := example.Projection["recovery"].(map[string]any)
			actualRecovery := actual["recovery"].(map[string]any)
			for _, key := range []string{"status", "inventoryComplete", "graphStatus", "candidates"} {
				if !reflect.DeepEqual(actualRecovery[key], expectedRecovery[key]) {
					t.Fatalf("recovery.%s: got %+v want %+v", key, actualRecovery[key], expectedRecovery[key])
				}
			}
			expectedComparison := example.Projection["comparison"].(map[string]any)
			actualComparison := actual["comparison"].(map[string]any)
			for _, key := range []string{"complete", "baselineAvailable"} {
				if actualComparison[key] != expectedComparison[key] {
					t.Fatalf("comparison.%s: got %+v want %+v", key, actualComparison[key], expectedComparison[key])
				}
			}
			if result.Complete != example.Projection["complete"] {
				t.Fatalf("complete: %v", result.Complete)
			}
			if _, ok := actual["context"].(map[string]any); !ok {
				t.Fatal("missing full context")
			}
			if _, ok := actual["orientation"].(map[string]any); !ok {
				t.Fatal("missing full orientation")
			}
		})
	}
}
func assertProjectionShape(t *testing.T, path string, actual, example any) {
	t.Helper()
	switch expected := example.(type) {
	case map[string]any:
		object, ok := actual.(map[string]any)
		if !ok {
			t.Fatalf("%s must be object: %#v", path, actual)
		}
		for key, value := range expected {
			found, exists := object[key]
			if !exists {
				t.Fatalf("%s missing required %s", path, key)
			}
			assertProjectionShape(t, path+"."+key, found, value)
		}
	case []any:
		array, ok := actual.([]any)
		if !ok {
			t.Fatalf("%s must be array: %#v", path, actual)
		}
		if len(array) != len(expected) {
			t.Fatalf("%s length got %d want %d", path, len(array), len(expected))
		}
		for index, value := range expected {
			assertProjectionShape(t, fmt.Sprintf("%s[%d]", path, index), array[index], value)
		}
	case nil:
		if actual != nil {
			t.Fatalf("%s must be explicit null: %#v", path, actual)
		}
	default:
		if reflect.TypeOf(actual) != reflect.TypeOf(example) {
			t.Fatalf("%s type got %T want %T", path, actual, example)
		}
	}
}
