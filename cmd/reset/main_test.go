package main

import (
	"go/ast"
	"go/parser"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple identifier",
			input:    "int",
			expected: "int",
		},
		{
			name:     "pointer type",
			input:    "*int",
			expected: "*int",
		},
		{
			name:     "slice type",
			input:    "[]string",
			expected: "[]string",
		},
		{
			name:     "map type",
			input:    "map[string]int",
			expected: "map[string]int",
		},
		{
			name:     "selector expression",
			input:    "time.Time",
			expected: "Time",
		},
		{
			name:     "pointer to selector",
			input:    "*time.Duration",
			expected: "*Duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.input)
			if err != nil {
				t.Fatalf("failed to parse expression: %v", err)
			}

			result := getTypeName(expr)
			if result != tt.expected {
				t.Errorf("getTypeName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"int", "int", "0"},
		{"int64", "int64", "0"},
		{"float32", "float32", "0"},
		{"float64", "float64", "0"},
		{"string", "string", `""`},
		{"bool", "bool", "false"},
		{"byte", "byte", "0"},
		{"rune", "rune", "0"},
		{"unknown type", "MyType", "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getZeroValue(tt.input)
			if result != tt.expected {
				t.Errorf("getZeroValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateFieldReset(t *testing.T) {
	tests := []struct {
		name     string
		field    fieldInfo
		expected []string
	}{
		{
			name: "basic int field",
			field: fieldInfo{
				name: "Count",
				typ:  &ast.Ident{Name: "int"},
			},
			expected: []string{"x.Count = 0"},
		},
		{
			name: "string field",
			field: fieldInfo{
				name: "Name",
				typ:  &ast.Ident{Name: "string"},
			},
			expected: []string{`x.Name = ""`},
		},
		{
			name: "bool field",
			field: fieldInfo{
				name: "Active",
				typ:  &ast.Ident{Name: "bool"},
			},
			expected: []string{"x.Active = false"},
		},
		{
			name: "slice field",
			field: fieldInfo{
				name: "Items",
				typ: &ast.ArrayType{
					Len: nil, // nil для слайса
					Elt: &ast.Ident{Name: "string"},
				},
			},
			expected: []string{"x.Items = x.Items[:0]"},
		},
		{
			name: "map field",
			field: fieldInfo{
				name: "Lookup",
				typ: &ast.MapType{
					Key:   &ast.Ident{Name: "string"},
					Value: &ast.Ident{Name: "int"},
				},
			},
			expected: []string{"clear(x.Lookup)"},
		},
		{
			name: "pointer to basic type",
			field: fieldInfo{
				name: "Value",
				typ: &ast.StarExpr{
					X: &ast.Ident{Name: "int"},
				},
			},
			expected: []string{
				"if x.Value != nil {",
				"*x.Value = 0",
				"}",
			},
		},
		{
			name: "array field",
			field: fieldInfo{
				name: "Fixed",
				typ: &ast.ArrayType{
					Len: &ast.BasicLit{Value: "10"}, // массив с размером
					Elt: &ast.Ident{Name: "int"},
				},
			},
			expected: []string{"// x.Fixed is array, skipping reset"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			generateFieldReset(&buf, "x", tt.field)
			result := buf.String()

			for _, expectedLine := range tt.expected {
				if !strings.Contains(result, expectedLine) {
					t.Errorf("generateFieldReset() output doesn't contain %q\ngot:\n%s", expectedLine, result)
				}
			}
		})
	}
}

func TestGenerateResetMethod(t *testing.T) {
	structInfo := structInfo{
		name: "User",
		fields: []fieldInfo{
			{name: "ID", typ: &ast.Ident{Name: "int"}},
			{name: "Name", typ: &ast.Ident{Name: "string"}},
			{name: "Active", typ: &ast.Ident{Name: "bool"}},
		},
	}

	result := generateResetMethod(structInfo)

	expectedLines := []string{
		"func (u *User) Reset() {",
		"    if u == nil {",
		"        return",
		"    }",
		"    u.ID = 0",
		`    u.Name = ""`,
		"    u.Active = false",
		"}",
	}

	for _, line := range expectedLines {
		if !strings.Contains(result, line) {
			t.Errorf("generateResetMethod() output doesn't contain %q\ngot:\n%s", line, result)
		}
	}
}

func TestAnalyzePackageWithResetComment(t *testing.T) {
	// Создаем временный каталог с тестовым Go файлом
	tmpDir := t.TempDir()

	testFile := `package testpkg

// generate:reset
type TestStruct struct {
	ID   int
	Name string
}

type AnotherStruct struct {
	Value float64
}
`

	filePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(filePath, []byte(testFile), 0644); err != nil {
		t.Fatal(err)
	}

	pkg := &pkgInfo{
		path:  tmpDir,
		files: []string{filePath},
	}

	if err := analyzePackage(pkg); err != nil {
		t.Fatalf("analyzePackage() error = %v", err)
	}

	if len(pkg.structs) != 1 {
		t.Fatalf("expected 1 struct with reset comment, got %d", len(pkg.structs))
	}

	if pkg.structs[0].name != "TestStruct" {
		t.Errorf("expected struct name 'TestStruct', got %q", pkg.structs[0].name)
	}

	if len(pkg.structs[0].fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(pkg.structs[0].fields))
	}
}

func TestAnalyzePackageWithoutResetComment(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package testpkg

type TestStruct struct {
	ID   int
	Name string
}
`

	filePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(filePath, []byte(testFile), 0644); err != nil {
		t.Fatal(err)
	}

	pkg := &pkgInfo{
		path:  tmpDir,
		files: []string{filePath},
	}

	if err := analyzePackage(pkg); err != nil {
		t.Fatalf("analyzePackage() error = %v", err)
	}

	if len(pkg.structs) != 0 {
		t.Errorf("expected 0 structs with reset comment, got %d", len(pkg.structs))
	}
}

func TestGenerateResetFile(t *testing.T) {
	tmpDir := t.TempDir()

	pkg := &pkgInfo{
		path:  tmpDir,
		name:  "testpkg",
		files: []string{"test.go"},
		structs: []structInfo{
			{
				name: "User",
				fields: []fieldInfo{
					{name: "ID", typ: &ast.Ident{Name: "int"}},
					{name: "Name", typ: &ast.Ident{Name: "string"}},
				},
			},
		},
	}

	if err := generateResetFile(pkg); err != nil {
		t.Fatalf("generateResetFile() error = %v", err)
	}

	outputPath := filepath.Join(tmpDir, "reset.gen.go")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedContent := []string{
		"// Code generated by reset tool. DO NOT EDIT.",
		"package testpkg",
		"func (u *User) Reset() {",
	}

	result := string(content)
	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("generated file doesn't contain %q", expected)
		}
	}
}
func TestGenerateResetFunctionsIntegration(t *testing.T) {
	// Создаем тестовую структуру каталогов
	tmpDir := t.TempDir()

	// Создаем подкаталог пакета
	pkgDir := filepath.Join(tmpDir, "testpkg")
	if err := os.Mkdir(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Файл с разными типами полей
	typesFile := `package testpkg

// generate:reset
type ComplexStruct struct {
	// Базовые типы
	IntVal    int
	StringVal string
	BoolVal   bool
	FloatVal  float64

	// Составные типы
	SliceVal   []string
	MapVal     map[string]int
	PointerVal *int

	// Вложенные структуры
	EmbeddedStruct
	
	// Указатель на структуру
	Nested *NestedStruct
}

type EmbeddedStruct struct {
	Data string
}

type NestedStruct struct {
	Value int
}

// generate:reset
type SimpleStruct struct {
	ID   int
	Name string
}
`

	// Дополнительный файл без помеченных структур
	extraFile := `package testpkg

type UnusedStruct struct {
	Temp float32
}
`

	// Записываем файлы
	if err := os.WriteFile(filepath.Join(pkgDir, "types.go"), []byte(typesFile), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "extra.go"), []byte(extraFile), 0644); err != nil {
		t.Fatal(err)
	}

	// Запускаем генерацию из родительского каталога
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := generateResetFunctions(tmpDir); err != nil {
		t.Fatalf("generateResetFunctions() error = %v", err)
	}

	// Проверяем сгенерированный файл
	genFile := filepath.Join(pkgDir, "reset.gen.go")
	content, err := os.ReadFile(genFile)
	if err != nil {
		t.Fatal(err)
	}

	genContent := string(content)

	// Проверяем наличие методов для обеих помеченных структур
	expectedStructs := []string{"ComplexStruct", "SimpleStruct"}
	for _, structName := range expectedStructs {
		receiver := strings.ToLower(string(structName[0]))
		methodSig := "func (" + receiver + " *" + structName + ") Reset()"
		if !strings.Contains(genContent, methodSig) {
			t.Errorf("missing Reset method for %s", structName)
		}
	}

	// Проверяем отсутствие метода для непомеченной структуры
	if strings.Contains(genContent, "func (") &&
		strings.Contains(genContent, "UnusedStruct") &&
		strings.Contains(genContent, "Reset()") {
		t.Error("should not generate reset method for unmarked struct")
	}

	// Проверяем правильность обработки разных типов полей
	// Для ComplexStruct receiver = "c"
	expectedPatterns := []string{
		"c.IntVal = 0",
		`c.StringVal = ""`,
		"c.BoolVal = false",
		"c.FloatVal = 0",
		"c.SliceVal = c.SliceVal[:0]",
		"clear(c.MapVal)",
		"if c.PointerVal != nil {",
		"*c.PointerVal = 0",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(genContent, pattern) {
			t.Errorf("generated code missing: %s", pattern)
			t.Logf("Generated content:\n%s", genContent)
		}
	}
}

func TestSkipVendorAndHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем структуру каталогов
	dirs := []string{
		filepath.Join(tmpDir, ".git"),
		filepath.Join(tmpDir, "vendor"),
		filepath.Join(tmpDir, ".hidden"),
		filepath.Join(tmpDir, "normal"),
	}

	for _, dir := range dirs {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Создаем файлы в каждом каталоге
	for _, dir := range dirs {
		file := filepath.Join(dir, "test.go")
		content := []byte("package test\n// generate:reset\ntype Test struct{}")
		if err := os.WriteFile(file, content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Запускаем анализ через WalkDir
	packages := make(map[string]*pkgInfo)
	err := filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем скрытые директории и vendor
		if d.IsDir() && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor") {
			return filepath.SkipDir
		}

		// Пропускаем тестовые файлы
		if !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			dir := filepath.Dir(path)
			if packages[dir] == nil {
				packages[dir] = &pkgInfo{
					path:  dir,
					files: []string{path},
				}
			} else {
				packages[dir].files = append(packages[dir].files, path)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("filepath.WalkDir error = %v", err)
	}

	// Проверяем, что скрытые директории и vendor пропущены
	for dir := range packages {
		if strings.Contains(dir, ".git") || strings.Contains(dir, ".hidden") || strings.Contains(dir, "vendor") {
			t.Errorf("should skip hidden/vendor directories, but found %q", dir)
		}
	}

	// Должен быть только normal каталог
	if len(packages) != 1 || !strings.Contains(packages[dirs[3]].path, "normal") {
		t.Errorf("expected only 'normal' directory, got %v", packages)
	}
}

func TestAnonymousFields(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package testpkg

// generate:reset
type Container struct {
	ID int
	time.Time  // анонимное поле
}
`

	filePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(filePath, []byte(testFile), 0644); err != nil {
		t.Fatal(err)
	}

	pkg := &pkgInfo{
		path:  tmpDir,
		files: []string{filePath},
	}

	if err := analyzePackage(pkg); err != nil {
		t.Fatalf("analyzePackage() error = %v", err)
	}

	if len(pkg.structs) != 1 {
		t.Fatalf("expected 1 struct, got %d", len(pkg.structs))
	}

	// Проверяем анонимные поля
	structInfo := pkg.structs[0]
	if len(structInfo.fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(structInfo.fields))
	}

	// Проверяем, что анонимные поля обработаны
	fieldNames := []string{"ID", "Time"}
	for i, field := range structInfo.fields {
		if field.name != fieldNames[i] {
			t.Errorf("field[%d].name = %q, want %q", i, field.name, fieldNames[i])
		}
	}
}

// Тест для проверки фильтрации тестовых файлов
func TestFilterTestFilesInWalkDir(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testpkg")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Создаем разные типы файлов
	files := map[string]string{
		"main.go":         "package testpkg\n// generate:reset\ntype Main struct{}",
		"main_test.go":    "package testpkg\n// generate:reset\ntype Test struct{}",
		"another.go":      "package testpkg\ntype Another struct{}",
		"another_test.go": "package testpkg\ntype AnotherTest struct{}",
	}

	for filename, content := range files {
		if err := os.WriteFile(filepath.Join(testDir, filename), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Запускаем обход директории как в основной функции
	packages := make(map[string]*pkgInfo)
	err := filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем скрытые директории и vendor
		if d.IsDir() && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor") {
			return fs.SkipDir
		}

		// Пропускаем тестовые файлы
		if !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			dir := filepath.Dir(path)
			if packages[dir] == nil {
				packages[dir] = &pkgInfo{
					path:  dir,
					files: []string{path},
				}
			} else {
				packages[dir].files = append(packages[dir].files, path)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("filepath.WalkDir error = %v", err)
	}

	// Проверяем, что только не-тестовые файлы включены
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	pkg := packages[testDir]
	if len(pkg.files) != 2 {
		t.Fatalf("expected 2 non-test files, got %d: %v", len(pkg.files), pkg.files)
	}

	// Проверяем, что файлы _test.go не включены
	for _, file := range pkg.files {
		if strings.HasSuffix(file, "_test.go") {
			t.Errorf("test file %q should not be included", file)
		}
	}
}
