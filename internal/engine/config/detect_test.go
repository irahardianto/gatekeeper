package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDetectStacks_Go(t *testing.T) {
	files := []string{"go.mod", "main.go", "README.md"}
	stacks := DetectStacks(files)

	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}
	if stacks[0] != StackGo {
		t.Errorf("expected Go stack, got %v", stacks[0])
	}
}

func TestDetectStacks_Node(t *testing.T) {
	files := []string{"package.json", "index.ts", "README.md"}
	stacks := DetectStacks(files)

	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}
	if stacks[0] != StackNode {
		t.Errorf("expected Node stack, got %v", stacks[0])
	}
}

func TestDetectStacks_Python_Requirements(t *testing.T) {
	files := []string{"requirements.txt", "app.py"}
	stacks := DetectStacks(files)

	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}
	if stacks[0] != StackPython {
		t.Errorf("expected Python stack, got %v", stacks[0])
	}
}

func TestDetectStacks_Python_Pyproject(t *testing.T) {
	files := []string{"pyproject.toml", "src/main.py"}
	stacks := DetectStacks(files)

	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}
	if stacks[0] != StackPython {
		t.Errorf("expected Python stack, got %v", stacks[0])
	}
}

func TestDetectStacks_Monorepo(t *testing.T) {
	files := []string{"go.mod", "package.json", "main.go", "index.ts"}
	stacks := DetectStacks(files)

	if len(stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d: %v", len(stacks), stacks)
	}

	found := map[Stack]bool{}
	for _, s := range stacks {
		found[s] = true
	}
	if !found[StackGo] || !found[StackNode] {
		t.Errorf("expected Go and Node stacks, got %v", stacks)
	}
}

func TestDetectStacks_AllThree(t *testing.T) {
	files := []string{"go.mod", "package.json", "requirements.txt"}
	stacks := DetectStacks(files)

	if len(stacks) != 3 {
		t.Fatalf("expected 3 stacks, got %d: %v", len(stacks), stacks)
	}
}

func TestDetectStacks_PythonNoDuplicates(t *testing.T) {
	// Both requirements.txt and pyproject.toml present — should only detect Python once.
	files := []string{"requirements.txt", "pyproject.toml", "app.py"}
	stacks := DetectStacks(files)

	pythonCount := 0
	for _, s := range stacks {
		if s == StackPython {
			pythonCount++
		}
	}
	if pythonCount != 1 {
		t.Errorf("expected 1 Python stack, got %d", pythonCount)
	}
}

func TestDetectStacks_None(t *testing.T) {
	files := []string{"README.md", "Makefile", "Dockerfile"}
	stacks := DetectStacks(files)

	if len(stacks) != 0 {
		t.Errorf("expected 0 stacks, got %d: %v", len(stacks), stacks)
	}
}

func TestDetectStacks_Empty(t *testing.T) {
	stacks := DetectStacks(nil)

	if len(stacks) != 0 {
		t.Errorf("expected 0 stacks for nil input, got %d", len(stacks))
	}
}

// --- GenerateGatesYAML Tests ---

func TestGenerateGatesYAML_Go(t *testing.T) {
	yaml := GenerateGatesYAML([]Stack{StackGo})

	assertYAMLContains(t, yaml, "version: 1")
	assertYAMLContains(t, yaml, "go vet")
	assertYAMLContains(t, yaml, "go test")
	assertYAMLContains(t, yaml, "golang")
}

func TestGenerateGatesYAML_Node(t *testing.T) {
	yaml := GenerateGatesYAML([]Stack{StackNode})

	assertYAMLContains(t, yaml, "version: 1")
	assertYAMLContains(t, yaml, "eslint")
	assertYAMLContains(t, yaml, "node")
}

func TestGenerateGatesYAML_Python(t *testing.T) {
	yaml := GenerateGatesYAML([]Stack{StackPython})

	assertYAMLContains(t, yaml, "version: 1")
	assertYAMLContains(t, yaml, "ruff")
	assertYAMLContains(t, yaml, "python")
}

func TestGenerateGatesYAML_NoStack(t *testing.T) {
	yaml := GenerateGatesYAML(nil)

	assertYAMLContains(t, yaml, "version: 1")
	// Should contain a commented example gate
	if !strings.Contains(yaml, "#") {
		t.Error("expected commented example in fallback config")
	}
}

func TestGenerateGatesYAML_Monorepo(t *testing.T) {
	yaml := GenerateGatesYAML([]Stack{StackGo, StackNode})

	assertYAMLContains(t, yaml, "go vet")
	assertYAMLContains(t, yaml, "eslint")
}

func TestGenerateGatesYAML_Parseable(t *testing.T) {
	// Verify generated YAML can be parsed back by our config types.
	for _, stacks := range [][]Stack{
		{StackGo},
		{StackNode},
		{StackPython},
		{StackGo, StackNode},
		{StackGo, StackNode, StackPython},
	} {
		yamlStr := GenerateGatesYAML(stacks)
		var cfg GatekeeperConfig
		err := yaml.Unmarshal([]byte(yamlStr), &cfg)
		if err != nil {
			t.Errorf("GenerateGatesYAML(%v) produced invalid YAML: %v\n%s", stacks, err, yamlStr)
		}
		if cfg.Version != 1 {
			t.Errorf("GenerateGatesYAML(%v) version = %d, want 1", stacks, cfg.Version)
		}
		if len(cfg.Gates) == 0 {
			t.Errorf("GenerateGatesYAML(%v) produced no gates", stacks)
		}
	}
}

func assertYAMLContains(t *testing.T, yaml, substr string) {
	t.Helper()
	if !strings.Contains(yaml, substr) {
		t.Errorf("expected generated YAML to contain %q, got:\n%s", substr, yaml)
	}
}
