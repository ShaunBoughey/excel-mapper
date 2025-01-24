# Go Onboarding Guide for Excel Mapper

## Prerequisites
1. Install Go from https://golang.org/dl/
2. Set up your Go workspace
3. Install an IDE with Go support (VS Code recommended with Go extension)

## Project Structure Overview
```
excel-mapper/
├── main.go           # Main application entry point
├── main_test.go      # Test files
├── config/           # Configuration files
├── docs/            # API documentation
└── uploads/         # File upload directory
```

## Key Go Concepts Used in This Project

### 1. HTTP Server
```go
// How we create and start the HTTP server
http.HandleFunc("/api/v1/process", handleAPIProcess)
http.ListenAndServe(":8080", nil)
```

### 2. Structs and Types
```go
// Example of struct usage in our project
type FieldConfigResponse struct {
    Fields []struct {
        Name        string `json:"name"`
        DisplayName string `json:"displayName"`
        IsMandatory bool   `json:"isMandatory"`
    }
    MandatoryFields []string `json:"mandatoryFields"`
}
```

### 3. File Handling
```go
// Example of file operations
file, handler, err := r.FormFile("file")
if err != nil {
    return err
}
defer file.Close()
```

### 4. JSON Handling
```go
// Parsing JSON
var fieldMappings map[string]string
if err := json.Unmarshal([]byte(mappingsStr), &fieldMappings); err != nil {
    return err
}
```

## Common Go Patterns Used

### 1. Error Handling
- Always check error returns
- Use `defer` for cleanup
- Return errors up the call stack

```go
f, err := os.Open(filename)
if err != nil {
    return fmt.Errorf("error opening file: %v", err)
}
defer f.Close()
```

### 2. HTTP Handlers
```go
func handleAPIProcess(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // ... handler logic
}
```

### 3. Testing
```go
func TestHandleUpload(t *testing.T) {
    // Create test request
    req := httptest.NewRequest(...)
    // Create response recorder
    rec := httptest.NewRecorder()
    // Call handler
    handler.ServeHTTP(rec, req)
    // Assert results
    if rec.Code != http.StatusOK {
        t.Errorf("expected status OK; got %v", rec.Code)
    }
}
```

## Key Dependencies

1. **excelize/v2**: Excel file handling
   ```go
   import "github.com/xuri/excelize/v2"
   ```

2. **http-swagger**: Swagger UI
   ```go
   import httpSwagger "github.com/swaggo/http-swagger"
   ```

## Common Tasks

### 1. Running the Project
```bash
go run main.go
```

### 2. Running Tests
```bash
go test ./...
```

### 3. Adding a New API Endpoint
1. Add route in `main.go`
2. Create handler function
3. Add Swagger documentation
4. Write tests

### 4. Working with Excel Files
```go
// Reading Excel files
f, err := excelize.OpenFile(filePath)
if err != nil {
    return nil, fmt.Errorf("error opening xlsx file: %v", err)
}
defer f.Close()

// Getting data from a sheet
rows, err := f.GetRows(sheetName)
```

## Debugging Tips
1. Use `fmt.Printf()` or `log.Printf()` for debugging
2. Run tests with `-v` flag for verbose output
3. Use Go's built-in race detector: `go test -race`

## Best Practices
1. Always handle errors
2. Use meaningful variable names
3. Write tests for new functionality
4. Document public functions
5. Use Go formatting: `go fmt`
6. Follow Go idioms and conventions

## Resources
1. [Go Documentation](https://golang.org/doc/)
2. [Go by Example](https://gobyexample.com/)
3. [Effective Go](https://golang.org/doc/effective_go)
4. [Project GitHub](https://github.com/ShaunBoughey/excel-mapper)

## Troubleshooting Development Issues

### 1. Build Errors

#### "package xxx is not in GOROOT"
- **Cause**: Missing dependency
- **Solution**: Run `go mod tidy` to install dependencies

#### "multiple modules in build list provide package xxx"
- **Cause**: Conflicting dependency versions
- **Solution**: Check `go.mod` and update dependencies:
  ```bash
  go mod why github.com/package-name
  go get -u github.com/package-name@latest
  ```

### 2. Runtime Errors

#### "panic: runtime error: invalid memory address or nil pointer dereference"
- **Cause**: Trying to use a nil pointer
- **Solution**: Add nil checks:
  ```go
  if myPointer != nil {
      myPointer.DoSomething()
  }
  ```

#### "panic: interface conversion"
- **Cause**: Invalid type assertion
- **Solution**: Use type assertion with ok check:
  ```go
  value, ok := interface{}.(Type)
  if !ok {
      // Handle error
  }
  ```

### 3. Testing Issues

#### "test failed with no output"
- **Cause**: Test timeout or panic
- **Solution**: Add `-v` flag and increase timeout:
  ```bash
  go test -v -timeout 30s ./...
  ```

#### "race detected during execution of test"
- **Cause**: Concurrent access to shared resources
- **Solution**: Use proper synchronization:
  ```go
  var mu sync.Mutex
  mu.Lock()
  defer mu.Unlock()
  ```

### 4. Common Excel Processing Issues

#### "cannot open file"
- **Cause**: File permissions or path issues
- **Solution**: Check file permissions and use absolute paths:
  ```go
  path, _ := filepath.Abs("./uploads/file.xlsx")
  f, err := excelize.OpenFile(path)
  ```

#### "sheet index out of range"
- **Cause**: Trying to access non-existent sheet
- **Solution**: Get sheet name first:
  ```go
  sheetName := f.GetSheetName(0)
  rows, err := f.GetRows(sheetName)
  ```

### 5. IDE Issues

#### Go extension not working in VS Code
1. Reload VS Code
2. Update Go tools:
   ```bash
   go install -v golang.org/x/tools/gopls@latest
   ```
3. Check Go extension settings

#### Debugging not working
1. Install Delve debugger:
   ```bash
   go install github.com/go-delve/delve/cmd/dlv@latest
   ```
2. Configure launch.json in VS Code
3. Set breakpoints and use F5 to debug

### 6. Common Go Commands for Troubleshooting
```bash
# Clean build cache
go clean -cache

# Show dependency graph
go mod graph

# Verify dependencies
go mod verify

# Format code
go fmt ./...

# Run tests with race detection
go test -race ./...

# Show package documentation
go doc package.name
```

### 7. Using Go's Built-in Profiling
```go
import _ "net/http/pprof"

// Add to main():
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Then use:
```bash
go tool pprof http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/profile
``` 