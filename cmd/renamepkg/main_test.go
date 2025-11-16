package main

import (
	"strings"
	"testing"
)

func TestReplaceImports(t *testing.T) {
	oldImport := "github.com/pillar/chrop/internal/server/di"
	newImport := "github.com/pillar/chrop/internal/server/difish"
	alias := "di"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single line import without alias",
			input: `package main

import "github.com/pillar/chrop/internal/server/di"

func main() {
}`,
			expected: `package main

import di "github.com/pillar/chrop/internal/server/difish"

func main() {
}`,
		},
		{
			name: "single line import with alias",
			input: `package main

import oldAlias "github.com/pillar/chrop/internal/server/di"

func main() {
}`,
			expected: `package main

import di "github.com/pillar/chrop/internal/server/difish"

func main() {
}`,
		},
		{
			name: "import block without alias",
			input: `package main

import (
	"github.com/pillar/chrop/internal/server/di"
	"other/package"
)

func main() {
}`,
			expected: `package main

import (
	di "github.com/pillar/chrop/internal/server/difish"
	"other/package"
)

func main() {
}`,
		},
		{
			name: "import block with alias",
			input: `package main

import (
	oldAlias "github.com/pillar/chrop/internal/server/di"
	"other/package"
)

func main() {
}`,
			expected: `package main

import (
	di "github.com/pillar/chrop/internal/server/difish"
	"other/package"
)

func main() {
}`,
		},
		{
			name: "import block multiple imports",
			input: `package main

import (
	"github.com/pillar/chrop/internal/server/di"
	"other/package1"
	"another/package2"
)

func main() {
}`,
			expected: `package main

import (
	di "github.com/pillar/chrop/internal/server/difish"
	"other/package1"
	"another/package2"
)

func main() {
}`,
		},
		{
			name: "mixed single line and block imports",
			input: `package main

import "github.com/pillar/chrop/internal/server/di"
import (
	"other/package"
)

func main() {
}`,
			expected: `package main

import di "github.com/pillar/chrop/internal/server/difish"
import (
	"other/package"
)

func main() {
}`,
		},
		{
			name: "import block with tab indentation",
			input: `package main

import (
		"github.com/pillar/chrop/internal/server/di"
)

func main() {
}`,
			expected: `package main

import (
		di "github.com/pillar/chrop/internal/server/difish"
)

func main() {
}`,
		},
		{
			name: "no matching import",
			input: `package main

import "other/package"

func main() {
}`,
			expected: `package main

import "other/package"

func main() {
}`,
		},
		{
			name: "import in comment should not be replaced",
			input: `package main

// import "github.com/pillar/chrop/internal/server/di"
import "other/package"

func main() {
}`,
			expected: `package main

// import "github.com/pillar/chrop/internal/server/di"
import "other/package"

func main() {
}`,
		},
		{
			name: "import block with existing alias and new alias",
			input: `package main

import (
	oldAlias "github.com/pillar/chrop/internal/server/di"
	"github.com/pillar/chrop/internal/server/di"
)

func main() {
}`,
			expected: `package main

import (
	di "github.com/pillar/chrop/internal/server/difish"
	di "github.com/pillar/chrop/internal/server/difish"
)

func main() {
}`,
		},
		{
			name: "complex import block",
			input: `package main

import (
	"fmt"
	"testing"
	
	"github.com/pillar/chrop/internal/server/di"
	
	"other/package"
)

func main() {
}`,
			expected: `package main

import (
	"fmt"
	"testing"
	
	di "github.com/pillar/chrop/internal/server/difish"
	
	"other/package"
)

func main() {
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test case needs alias because di -> difish (last segment changed)
			result := replaceImports(tt.input, oldImport, newImport, alias, true)
			if normalize(result) != normalize(tt.expected) {
				t.Errorf("replaceImports() = \n%v\n, want \n%v", result, tt.expected)
			}
		})
	}
}

func TestReplaceImportsWithoutAlias(t *testing.T) {
	// Test case where last segment doesn't change, so no alias needed
	oldImport := "github.com/pillar/chrop/internal/server/di"
	newImport := "github.com/pillar/chrop/internal/app/di"
	alias := "di"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single line import without alias - should not add alias",
			input: `package main

import "github.com/pillar/chrop/internal/server/di"

func main() {
}`,
			expected: `package main

import "github.com/pillar/chrop/internal/app/di"

func main() {
}`,
		},
		{
			name: "import block without alias - should not add alias",
			input: `package main

import (
	"github.com/pillar/chrop/internal/server/di"
	"other/package"
)

func main() {
}`,
			expected: `package main

import (
	"github.com/pillar/chrop/internal/app/di"
	"other/package"
)

func main() {
}`,
		},
		{
			name: "import block with existing alias - should keep alias",
			input: `package main

import (
	di "github.com/pillar/chrop/internal/server/di"
	"other/package"
)

func main() {
}`,
			expected: `package main

import (
	di "github.com/pillar/chrop/internal/app/di"
	"other/package"
)

func main() {
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// needAlias is false because di -> di (last segment unchanged)
			result := replaceImports(tt.input, oldImport, newImport, alias, false)
			if normalize(result) != normalize(tt.expected) {
				t.Errorf("replaceImports() = \n%v\n, want \n%v", result, tt.expected)
			}
		})
	}
}

func TestReplaceModuleImports(t *testing.T) {
	oldModule := "github.com/pillar/chrop"
	newModule := "github.com/pillar/doaddon"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single line import without alias - should not add alias",
			input: `package main

import "github.com/pillar/chrop/internal/server/di"

func main() {
}`,
			expected: `package main

import "github.com/pillar/doaddon/internal/server/di"

func main() {
}`,
		},
		{
			name: "single line import with alias - should keep alias",
			input: `package main

import di "github.com/pillar/chrop/internal/server/di"

func main() {
}`,
			expected: `package main

import di "github.com/pillar/doaddon/internal/server/di"

func main() {
}`,
		},
		{
			name: "import block without alias - should not add alias",
			input: `package main

import (
	"github.com/pillar/chrop/internal/server/di"
	"other/package"
)

func main() {
}`,
			expected: `package main

import (
	"github.com/pillar/doaddon/internal/server/di"
	"other/package"
)

func main() {
}`,
		},
		{
			name: "import block with alias - should keep alias",
			input: `package main

import (
	di "github.com/pillar/chrop/internal/server/di"
	"other/package"
)

func main() {
}`,
			expected: `package main

import (
	di "github.com/pillar/doaddon/internal/server/di"
	"other/package"
)

func main() {
}`,
		},
		{
			name: "multiple imports with and without alias",
			input: `package main

import (
	"github.com/pillar/chrop/internal/server/di"
	di2 "github.com/pillar/chrop/internal/server/di2"
	"other/package"
)

func main() {
}`,
			expected: `package main

import (
	"github.com/pillar/doaddon/internal/server/di"
	di2 "github.com/pillar/doaddon/internal/server/di2"
	"other/package"
)

func main() {
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceModuleImports(tt.input, oldModule, newModule)
			if normalize(result) != normalize(tt.expected) {
				t.Errorf("replaceModuleImports() = \n%v\n, want \n%v", result, tt.expected)
			}
		})
	}
}

// normalize removes trailing whitespace and normalizes line endings
func normalize(s string) string {
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		normalized = append(normalized, strings.TrimRight(line, " \t"))
	}
	return strings.Join(normalized, "\n")
}
