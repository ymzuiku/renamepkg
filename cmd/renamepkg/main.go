package main

import (
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

const version = "0.0.1"

// replaceImports uses regex to replace imports
// Replaces "oldImport" with "newImport", optionally adding alias if needAlias is true
func replaceImports(src, oldImport, newImport, alias string, needAlias bool) string {
	updated := src
	// Escape special regex characters in oldImport
	escapedOldImport := regexp.QuoteMeta(oldImport)
	oldImportQuoted := `"` + oldImport + `"`

	// Determine the replacement format based on whether alias is needed
	var newImportReplacement string
	if needAlias {
		newImportReplacement = fmt.Sprintf(`%s "%s"`, alias, newImport)
	} else {
		newImportReplacement = fmt.Sprintf(`"%s"`, newImport)
	}

	// Check if the import exists in the file
	if !strings.Contains(updated, oldImportQuoted) {
		return updated
	}

	// Process line by line to handle import blocks correctly
	lines := strings.Split(updated, "\n")
	inImportBlock := false
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect import block start
		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
		} else if inImportBlock && trimmed == ")" {
			inImportBlock = false
		}

		// Check if this line contains the old import path
		if strings.Contains(line, oldImportQuoted) {
			// Skip comments
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}

			if inImportBlock {
				// In import block: match "oldImport" or alias "oldImport"
				// Pattern: optional alias, then the import path
				linePattern := regexp.MustCompile(`(\s*)(\w+\s+)?"` + escapedOldImport + `"`)
				lines[i] = linePattern.ReplaceAllStringFunc(line, func(match string) string {
					submatches := linePattern.FindStringSubmatch(match)
					if len(submatches) >= 3 {
						indent := submatches[1]
						existingAlias := strings.TrimSpace(submatches[2])

						if needAlias {
							// Always use new alias when needAlias is true
							return fmt.Sprintf(`%s%s`, indent, newImportReplacement)
						} else {
							// When needAlias is false, keep existing alias if present
							if existingAlias != "" {
								return fmt.Sprintf(`%s%s "%s"`, indent, existingAlias, newImport)
							}
							// No existing alias, just replace path
							return fmt.Sprintf(`%s"%s"`, indent, newImport)
						}
					}
					return match
				})
				modified = true
			} else {
				// Single line import: import "oldImport" or import alias "oldImport"
				// Pattern 1: without alias - must start with "import"
				pattern1 := regexp.MustCompile(`^(\s*)import\s+"` + escapedOldImport + `"`)
				if pattern1.MatchString(line) {
					lines[i] = pattern1.ReplaceAllStringFunc(line, func(match string) string {
						submatches := pattern1.FindStringSubmatch(match)
						return submatches[1] + "import " + newImportReplacement
					})
					modified = true
				} else {
					// Pattern 2: with alias - must start with "import"
					pattern2 := regexp.MustCompile(`^(\s*)import\s+(\w+\s+)"` + escapedOldImport + `"`)
					if pattern2.MatchString(line) {
						lines[i] = pattern2.ReplaceAllStringFunc(line, func(match string) string {
							submatches := pattern2.FindStringSubmatch(match)
							existingAlias := strings.TrimSpace(submatches[2])

							if needAlias {
								// Always use new alias when needAlias is true
								return submatches[1] + "import " + newImportReplacement
							} else {
								// When needAlias is false, keep existing alias
								return fmt.Sprintf(`%simport %s "%s"`, submatches[1], existingAlias, newImport)
							}
						})
						modified = true
					}
				}
			}
		}
	}

	if modified {
		updated = strings.Join(lines, "\n")
	}

	return updated
}

// replaceModuleImports replaces all imports that start with oldModule with newModule
// Preserves existing aliases, but does not add aliases if none exist
func replaceModuleImports(src, oldModule, newModule string) string {
	updated := src
	escapedOldModule := regexp.QuoteMeta(oldModule)

	// Check if any imports with oldModule exist
	if !strings.Contains(updated, oldModule) {
		return updated
	}

	// Pattern to match imports starting with oldModule
	// Matches: "oldModule/..." or alias "oldModule/..."
	lines := strings.Split(updated, "\n")
	inImportBlock := false
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect import block start
		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
		} else if inImportBlock && trimmed == ")" {
			inImportBlock = false
		}

		// Check if this line contains an import with oldModule
		if strings.Contains(line, oldModule) {
			// Pattern 1: Single line import without alias: import "oldModule/..."
			pattern1 := regexp.MustCompile(`(\s+import\s+)"(` + escapedOldModule + `/[^"]+)"`)
			if pattern1.MatchString(line) {
				line = pattern1.ReplaceAllStringFunc(line, func(match string) string {
					submatches := pattern1.FindStringSubmatch(match)
					if len(submatches) < 3 {
						return match
					}
					oldImportPath := submatches[2]
					newImportPath := strings.Replace(oldImportPath, oldModule, newModule, 1)
					// Don't add alias for module rename, just replace the path
					return fmt.Sprintf(`%s"%s"`, submatches[1], newImportPath)
				})
				modified = true
			}

			// Pattern 2: Single line import with alias: import alias "oldModule/..."
			pattern2 := regexp.MustCompile(`(\s+import\s+)(\w+\s+)"(` + escapedOldModule + `/[^"]+)"`)
			if pattern2.MatchString(line) {
				line = pattern2.ReplaceAllStringFunc(line, func(match string) string {
					submatches := pattern2.FindStringSubmatch(match)
					if len(submatches) < 4 {
						return match
					}
					alias := submatches[2]
					oldImportPath := submatches[3]
					newImportPath := strings.Replace(oldImportPath, oldModule, newModule, 1)
					return fmt.Sprintf(`%s%s "%s"`, submatches[1], alias, newImportPath)
				})
				modified = true
			}

			// Pattern 3: In import block: "oldModule/..." or alias "oldModule/..."
			if inImportBlock || strings.Contains(line, "import") {
				// Match: optional alias, then "oldModule/..."
				pattern3 := regexp.MustCompile(`(\s*)(\w+\s+)?"(` + escapedOldModule + `/[^"]+)"`)
				if pattern3.MatchString(line) {
					line = pattern3.ReplaceAllStringFunc(line, func(match string) string {
						submatches := pattern3.FindStringSubmatch(match)
						if len(submatches) < 4 {
							return match
						}
						indent := submatches[1]
						existingAlias := strings.TrimSpace(submatches[2])
						oldImportPath := submatches[3]
						newImportPath := strings.Replace(oldImportPath, oldModule, newModule, 1)

						if existingAlias != "" {
							// Keep existing alias
							return fmt.Sprintf(`%s%s "%s"`, indent, existingAlias, newImportPath)
						}

						// Don't add alias for module rename, just replace the path
						return fmt.Sprintf(`%s"%s"`, indent, newImportPath)
					})
					modified = true
				}
			}
		}

		lines[i] = line
	}

	if modified {
		updated = strings.Join(lines, "\n")
	}

	return updated
}

// readModuleFromGoMod reads the module path from go.mod file
func readModuleFromGoMod() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			// Extract module path after "module "
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}

// updateGoMod updates the module path in go.mod file
func updateGoMod(newModule string) error {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			// Replace module path
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				// Preserve the original indentation and format
				indent := ""
				for j, char := range line {
					if char != ' ' && char != '\t' {
						indent = line[:j]
						break
					}
				}
				lines[i] = indent + "module " + newModule
				modified = true
				break
			}
		}
	}

	if !modified {
		return fmt.Errorf("module declaration not found in go.mod")
	}

	updated := strings.Join(lines, "\n")
	return os.WriteFile("go.mod", []byte(updated), 0644)
}

// renameModule renames all imports from oldModule to newModule across all .go files
func renameModule(oldModule, newModule string) {
	oldModuleSlash := filepath.ToSlash(oldModule)
	newModuleSlash := filepath.ToSlash(newModule)

	fmt.Println("Rename module imports:")
	fmt.Println("  ", oldModuleSlash, "→", newModuleSlash)

	// Search all .go files in the project directory and replace import statements
	var filesProcessed, filesModified int
	err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip files in vendor, node_modules, .git directories
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "/node_modules/") || strings.Contains(path, "/.git/") {
			return nil
		}

		filesProcessed++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		originalContent := string(data)
		updated := replaceModuleImports(originalContent, oldModuleSlash, newModuleSlash)
		modified := updated != originalContent

		if modified {
			filesModified++
			fmt.Printf("  Updated: %s\n", path)
		}

		// gofmt
		formatted, err := format.Source([]byte(updated))
		if err != nil {
			fmt.Printf("Warning: cannot format %s: %v\n", path, err)
			formatted = []byte(updated)
		}

		return os.WriteFile(path, formatted, info.Mode())
	})

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Update go.mod file with new module path
	if err := updateGoMod(newModuleSlash); err != nil {
		fmt.Printf("Warning: failed to update go.mod: %v\n", err)
	} else {
		fmt.Printf("  Updated: go.mod\n")
	}

	fmt.Printf("\nCompleted successfully. Processed %d files, modified %d files.\n", filesProcessed, filesModified)
}

func renamePackageAction(c *cli.Context) error {
	from := c.String("from")
	to := c.String("to")
	mod := c.String("module")
	force := c.Bool("force")

	if from == "" || to == "" {
		return cli.Exit("Error: -from and -to are required", 1)
	}

	// Read module from go.mod if not provided
	var modulePath string
	if mod == "" {
		var err error
		modulePath, err = readModuleFromGoMod()
		if err != nil {
			return cli.Exit(fmt.Sprintf("Error: %v", err), 1)
		}
	} else {
		modulePath = mod
	}

	// Extract package name from 'from' path (e.g., "di" from "internal/server/di")
	// Only need alias if the last segment of the path changed
	fromBase := filepath.Base(from)
	toBase := filepath.Base(to)
	needAlias := fromBase != toBase
	alias := fromBase

	oldFullPath := filepath.Join(from)
	newFullPath := filepath.Join(to)

	// Check if target directory exists
	if _, err := os.Stat(newFullPath); err == nil {
		if force {
			fmt.Printf("Target directory %s exists, removing it (--force enabled)...\n", newFullPath)
			if err := os.RemoveAll(newFullPath); err != nil {
				return cli.Exit(fmt.Sprintf("Failed to remove target directory: %v", err), 1)
			}
		} else {
			return cli.Exit(fmt.Sprintf("Error: Target directory %s already exists.\nUse -force to overwrite it.", newFullPath), 1)
		}
	}

	// Ensure parent directories exist for the target path
	newFullPathParent := filepath.Dir(newFullPath)
	if err := os.MkdirAll(newFullPathParent, 0755); err != nil {
		return cli.Exit(fmt.Sprintf("Failed to create parent directories for %s: %v", newFullPath, err), 1)
	}

	// Step 1: rename folder
	if err := os.Rename(oldFullPath, newFullPath); err != nil {
		return cli.Exit(fmt.Sprintf("Failed to rename folder: %v", err), 1)
	}

	// Build import paths
	// Construct full import paths: modulePath/from -> modulePath/to
	modSlash := filepath.ToSlash(modulePath)
	fromSlash := filepath.ToSlash(from)
	toSlash := filepath.ToSlash(to)

	// Build full import paths
	oldImport := modSlash + "/" + fromSlash
	newImport := modSlash + "/" + toSlash

	fmt.Println("Rename import:")
	if needAlias {
		fmt.Printf("  \"%s\" → %s \"%s\"\n", oldImport, alias, newImport)
	} else {
		fmt.Printf("  \"%s\" → \"%s\"\n", oldImport, newImport)
	}

	// Step 2: Search all .go files in the project directory (execution directory, not package directory)
	// and replace import statements
	var filesProcessed, filesModified int
	err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip files in vendor, node_modules, .git directories
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "/node_modules/") || strings.Contains(path, "/.git/") {
			return nil
		}

		filesProcessed++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		originalContent := string(data)
		updated := originalContent
		modified := false

		// Replace import using regex
		updatedContent := replaceImports(originalContent, oldImport, newImport, alias, needAlias)
		if updatedContent != originalContent {
			updated = updatedContent
			modified = true
		}

		// If inside the new package dir, update `package xxx`
		if strings.HasPrefix(path, newFullPath) {
			oldPkg := filepath.Base(from)
			newPkg := filepath.Base(to)
			// Use regex to replace package declaration
			packagePattern := regexp.MustCompile(`^package\s+` + regexp.QuoteMeta(oldPkg) + `\s*$`)
			lines := strings.Split(updated, "\n")
			for i, line := range lines {
				if packagePattern.MatchString(strings.TrimSpace(line)) {
					lines[i] = strings.Replace(line, "package "+oldPkg, "package "+newPkg, 1)
					modified = true
					break
				}
			}
			updated = strings.Join(lines, "\n")
		}

		if modified {
			filesModified++
			fmt.Printf("  Updated: %s\n", path)
		}

		// gofmt
		formatted, err := format.Source([]byte(updated))
		if err != nil {
			fmt.Printf("Warning: cannot format %s: %v\n", path, err)
			formatted = []byte(updated)
		}

		return os.WriteFile(path, formatted, info.Mode())
	})

	if err != nil {
		return cli.Exit(fmt.Sprintf("Error: %v", err), 1)
	}

	fmt.Printf("\nCompleted successfully. Processed %d files, modified %d files.\n", filesProcessed, filesModified)

	// Only show alias refactoring hint if alias is needed
	if needAlias {
		fmt.Printf("\nPlease search for: %s \"%s\"\n", alias, newImport)
		fmt.Printf("Then use F2 to refactor the alias '%s'.\n", alias)
	}

	return nil
}

func renameModuleAction(c *cli.Context) error {
	newMod := c.String("mod")

	if newMod == "" {
		return cli.Exit("Error: -mod is required", 1)
	}

	// Read old module from go.mod
	oldMod, err := readModuleFromGoMod()
	if err != nil {
		return cli.Exit(fmt.Sprintf("Error: %v", err), 1)
	}

	renameModule(oldMod, newMod)
	return nil
}

func main() {
	app := &cli.App{
		Name:    "renamepkg",
		Usage:   "Rename Go packages and modules",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "mod",
				Aliases: []string{"m"},
				Usage:   "new module path (e.g. github.com/pillar/doaddon)",
			},
			&cli.StringFlag{
				Name:    "from",
				Aliases: []string{"f"},
				Usage:   "old import path (e.g. internal/server/di)",
			},
			&cli.StringFlag{
				Name:    "to",
				Aliases: []string{"t"},
				Usage:   "new import path (e.g. internal/server/difish)",
			},
			&cli.StringFlag{
				Name:  "module",
				Usage: "module path in go.mod (optional, will be read from go.mod if not provided)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"F"},
				Usage:   "force: delete target directory if it exists",
			},
		},
		Action: func(c *cli.Context) error {
			// Check if we're in module rename mode or package rename mode
			if c.String("mod") != "" {
				// Module rename mode: only -mod is required, old module is read from go.mod
				if c.String("from") != "" || c.String("to") != "" {
					return cli.Exit("Error: Cannot use -from/-to with -mod\nUsage: renamepkg -mod github.com/pillar/doaddon", 1)
				}
				return renameModuleAction(c)
			}

			// Package rename mode: -from, -to (module is read from go.mod if not provided)
			return renamePackageAction(c)
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
