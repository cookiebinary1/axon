package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/axon/pkg/fsctx"
	"github.com/axon/pkg/project"
)

// FileInfo represents information about an indexed file
type FileInfo struct {
	Path        string   `json:"path"`
	Size        int64    `json:"size"`
	Extension   string   `json:"extension"`
	IsDir       bool     `json:"is_dir"`
	Classes     []string `json:"classes,omitempty"`
	Functions   []string `json:"functions,omitempty"`
	Symbols     []Symbol `json:"symbols,omitempty"`
}

// Symbol represents a code symbol (class, function, etc.)
type Symbol struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // "class", "function", "method"
	Line      int    `json:"line"`
	Signature string `json:"signature,omitempty"`
}

// Index represents the project index
type Index struct {
	projectRoot string
	cfg         *project.Config
	files       map[string]*FileInfo // path -> FileInfo
	tree        *TreeNode
	mu          sync.RWMutex
}

// TreeNode represents a directory tree node
type TreeNode struct {
	Name     string               `json:"name"`
	Path     string               `json:"path"`
	IsDir    bool                 `json:"is_dir"`
	Children map[string]*TreeNode `json:"children,omitempty"`
	Files    []string             `json:"files,omitempty"`
}

// NewIndex creates a new project index
func NewIndex(projectRoot string, cfg *project.Config) *Index {
	return &Index{
		projectRoot: projectRoot,
		cfg:         cfg,
		files:       make(map[string]*FileInfo),
	}
}

// IndexProject indexes the entire project
func (idx *Index) IndexProject() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Clear existing index
	idx.files = make(map[string]*FileInfo)
	idx.tree = &TreeNode{
		Name:     filepath.Base(idx.projectRoot),
		Path:     "",
		IsDir:    true,
		Children: make(map[string]*TreeNode),
	}

	return filepath.Walk(idx.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Get relative path
		relPath, err := filepath.Rel(idx.projectRoot, path)
		if err != nil {
			return nil
		}

		// Normalize path
		normalizedPath := strings.ReplaceAll(relPath, "\\", "/")

		// Skip root itself
		if normalizedPath == "." {
			return nil
		}

		// Check if should ignore
		if idx.shouldIgnore(normalizedPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Index file or directory
		fileInfo := &FileInfo{
			Path:      normalizedPath,
			Size:      info.Size(),
			Extension: fsctx.GetFileExtension(path),
			IsDir:     info.IsDir(),
		}

		// If it's a code file, extract symbols
		if !info.IsDir() && idx.isCodeFile(path) {
			symbols := idx.extractSymbols(path, normalizedPath)
			fileInfo.Symbols = symbols
			fileInfo.Classes = extractClassNames(symbols)
			fileInfo.Functions = extractFunctionNames(symbols)
		}

		idx.files[normalizedPath] = fileInfo

		// Build tree structure
		idx.addToTree(normalizedPath, fileInfo)

		return nil
	})
}

// shouldIgnore checks if a path should be ignored
func (idx *Index) shouldIgnore(path string) bool {
	// Normalize path separators
	normalizedPath := strings.ReplaceAll(path, "\\", "/")
	return project.ShouldIgnore(normalizedPath, idx.cfg.Context.Ignore)
}

// isCodeFile checks if a file is a known code file type
func (idx *Index) isCodeFile(path string) bool {
	ext := fsctx.GetFileExtension(path)
	codeExtensions := map[string]bool{
		".go":   true,
		".php":  true,
		".js":   true,
		".ts":   true,
		".jsx":  true,
		".tsx":  true,
		".c":    true,
		".cpp":  true,
		".cc":   true,
		".cxx":  true,
		".h":    true,
		".hpp":  true,
		".lua":  true,
		".py":   true,
		".java": true,
		".rb":   true,
		".rs":   true,
		".swift": true,
		".kt":   true,
		".scala": true,
		".cs":   true,
		".dart": true,
		".sh":   true,
		".bash": true,
		".zsh":  true,
	}
	return codeExtensions[ext]
}

// extractSymbols extracts code symbols (classes, functions) from a file
func (idx *Index) extractSymbols(fullPath, relPath string) []Symbol {
	ext := fsctx.GetFileExtension(fullPath)
	
	switch ext {
	case ".go":
		return parseGoFile(fullPath)
	case ".php":
		return parsePHPFile(fullPath)
	case ".js", ".jsx":
		return parseJavaScriptFile(fullPath)
	case ".ts", ".tsx":
		return parseTypeScriptFile(fullPath)
	case ".c", ".cpp", ".cc", ".cxx", ".h", ".hpp":
		return parseCFile(fullPath)
	case ".lua":
		return parseLuaFile(fullPath)
	case ".py":
		return parsePythonFile(fullPath)
	case ".java":
		return parseJavaFile(fullPath)
	case ".rb":
		return parseRubyFile(fullPath)
	case ".rs":
		return parseRustFile(fullPath)
	case ".sh", ".bash", ".zsh":
		return parseShellFile(fullPath)
	default:
		return []Symbol{}
	}
}

// addToTree adds a file to the tree structure
func (idx *Index) addToTree(path string, fileInfo *FileInfo) {
	parts := strings.Split(path, "/")
	current := idx.tree

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - it's the file/dir itself
			if fileInfo.IsDir {
				if current.Children[part] == nil {
					current.Children[part] = &TreeNode{
						Name:     part,
						Path:     path,
						IsDir:    true,
						Children: make(map[string]*TreeNode),
						Files:    []string{},
					}
				}
			} else {
				if current.Files == nil {
					current.Files = []string{}
				}
				current.Files = append(current.Files, path)
			}
		} else {
			// Intermediate directory
				if current.Children[part] == nil {
					current.Children[part] = &TreeNode{
						Name:     part,
						Path:     strings.Join(parts[:i+1], "/"),
						IsDir:    true,
						Children: make(map[string]*TreeNode),
						Files:    []string{},
					}
				}
			current = current.Children[part]
		}
	}
}

// GetFileInfo returns information about a file
func (idx *Index) GetFileInfo(path string) (*FileInfo, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	
	info, ok := idx.files[path]
	return info, ok
}

// GetTree returns the tree structure starting from a path
func (idx *Index) GetTree(path string) (*TreeNode, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if path == "" || path == "." {
		return idx.tree, nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := idx.tree

	for _, part := range parts {
		if part == "" {
			continue
		}
		if current.Children[part] == nil {
			return nil, fmt.Errorf("path not found: %s", path)
		}
		current = current.Children[part]
	}

	return current, nil
}

// GetFileSymbols returns symbols (classes, functions) from a file
func (idx *Index) GetFileSymbols(path string) ([]Symbol, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	fileInfo, ok := idx.files[path]
	if !ok {
		return nil, fmt.Errorf("file not found in index: %s", path)
	}

	return fileInfo.Symbols, nil
}

// GetAllFilePaths returns all indexed file paths
func (idx *Index) GetAllFilePaths() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	paths := make([]string, 0, len(idx.files))
	for path := range idx.files {
		paths = append(paths, path)
	}
	return paths
}

// Helper functions to extract class and function names
func extractClassNames(symbols []Symbol) []string {
	classes := []string{}
	for _, sym := range symbols {
		if sym.Type == "class" || sym.Type == "interface" || sym.Type == "struct" {
			classes = append(classes, sym.Name)
		}
	}
	return classes
}

func extractFunctionNames(symbols []Symbol) []string {
	functions := []string{}
	for _, sym := range symbols {
		if sym.Type == "function" || sym.Type == "method" {
			functions = append(functions, sym.Name)
		}
	}
	return functions
}

