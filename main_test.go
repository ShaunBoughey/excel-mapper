package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"import/auth"

	"github.com/xuri/excelize/v2"
)

func init() {
	// Set test API key
	os.Setenv("API_KEYS", "test-api-key-1,test-api-key-2")
}

func TestServeUI(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	http.HandlerFunc(serveUI).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleUploadInvalidMethod(t *testing.T) {
	req, err := http.NewRequest("GET", "/upload", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleUpload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHandleUploadCSVFile(t *testing.T) {
	fileContent := `Account Number,Account Active,Customer Name,Customer ID
	1234,Yes,John Doe,1001
	2345,No,Jane Smith,1002`

	// Create a temporary file to simulate an uploaded file
	tempFile, err := os.CreateTemp("./uploads", "test_upload_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(fileContent)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Create a multipart form file
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile("fileInput", filepath.Base(tempFile.Name()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(part, tempFile)
	if err != nil {
		t.Fatal(err)
	}

	// Add other form fields
	_ = writer.WriteField("mapping_Account Number", "Account Number")
	_ = writer.WriteField("mapping_Account Active", "Account Active")
	_ = writer.WriteField("mapping_Customer Name", "Customer Name")
	_ = writer.WriteField("mapping_Customer ID", "Customer ID")

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleUpload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Validate response structure
	if success, ok := response["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	if _, ok := response["summary"].(string); !ok {
		t.Errorf("Expected summary field in response")
	}

	if _, ok := response["outputFilename"].(string); !ok {
		t.Errorf("Expected outputFilename field in response")
	}
}

func TestHandleUploadInvalidFileFormat(t *testing.T) {
	fileContent := `This is a plain text file, not a CSV or Excel file.`

	// Create a temporary file to simulate an uploaded file
	tempFile, err := os.CreateTemp("./uploads", "test_upload_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(fileContent)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Create a multipart form file
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile("fileInput", filepath.Base(tempFile.Name()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(part, tempFile)
	if err != nil {
		t.Fatal(err)
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleUpload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for invalid file: got %v want %v", status, http.StatusBadRequest)
	}

	if !strings.Contains(recorder.Body.String(), "Invalid file type. Only .csv and .xlsx files are allowed") {
		t.Errorf("handler did not indicate invalid file format: got %v", recorder.Body.String())
	}
}

func TestHandleDownload(t *testing.T) {
	// Update the file path to match the expected format without the leading "./uploads/"
	req, err := http.NewRequest("GET", "/download?file=processed_data.xlsx", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleDownload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check if the content type is correct
	expectedContentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if contentType := recorder.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}
}

func TestHandleUploadNoFile(t *testing.T) {
	// Test case where no file is uploaded
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add other form fields
	_ = writer.WriteField("mapping_Account Number", "Account Number")
	_ = writer.WriteField("mapping_Account Active", "Account Active")

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleUpload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for missing file: got %v want %v", status, http.StatusBadRequest)
	}

	if !strings.Contains(recorder.Body.String(), "No file uploaded") {
		t.Errorf("handler did not indicate missing file: got %v", recorder.Body.String())
	}
}

func TestHandleDownloadMissingFileParameter(t *testing.T) {
	// Test case where file parameter is missing
	req, err := http.NewRequest("GET", "/download", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleDownload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for missing file parameter: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestHandleDownloadNonExistentFile(t *testing.T) {
	// Test case where requested file does not exist
	req, err := http.NewRequest("GET", "/download?file=non_existent_file.xlsx", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleDownload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code for non-existent file: got %v want %v", status, http.StatusNotFound)
	}
}

func TestHandleDownloadInvalidFilePath(t *testing.T) {
	// Test case where requested file path is invalid (attempting path traversal)
	req, err := http.NewRequest("GET", "/download?file=../secret_file.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleDownload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for invalid file path: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestProcessFileSuccess(t *testing.T) {
	// Create a temporary Excel file for testing
	tempFile, err := os.CreateTemp("./uploads", "test_process_*.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	excelFile := excelize.NewFile()
	sheetName := "Sheet1"
	excelFile.SetSheetName("Sheet1", sheetName)

	// Add headers and some data to the file
	headers := []string{"Account Number", "Account Active", "Customer Name", "Customer ID"}
	for i, header := range headers {
		cell := string(rune('A'+i)) + "1"
		excelFile.SetCellValue(sheetName, cell, header)
	}

	dataRows := [][]string{{"1234", "Yes", "John Doe", "1001"}, {"2345", "No", "Jane Smith", "1002"}}
	for rowIndex, row := range dataRows {
		for colIndex, value := range row {
			cell := string(rune('A'+colIndex)) + string(rune('2'+rowIndex))
			excelFile.SetCellValue(sheetName, cell, value)
		}
	}

	if err := excelFile.SaveAs(tempFile.Name()); err != nil {
		t.Fatal(err)
	}

	fieldMappings := map[string]string{
		"Client Code":    "Account Number",
		"Customer ID":    "Customer ID",
		"Account Number": "Account Number",
	}
	order := []string{"Client Code", "Customer ID", "Account Number"}
	outputFormat := "excel"
	uniqueID := "test_" + generateUniqueID()
	summary, errStr := processFile(tempFile.Name(), fieldMappings, order, outputFormat, uniqueID)

	if errStr != "" && !strings.Contains(errStr, "processed_data.xlsx") {
		t.Errorf("unexpected error string: got %v", errStr)
	}

	if summary == "" {
		t.Errorf("unexpected empty summary")
	}
}

func TestProcessFileInvalidFile(t *testing.T) {
	invalidFilePath := "invalid/path/to/nonexistent_file.xlsx"

	fieldMappings := map[string]string{
		"Client Code":    "Account Number",
		"Customer ID":    "Customer ID",
		"Account Number": "Account Number",
	}
	order := []string{"Client Code", "Customer ID", "Account Number"}
	outputFormat := "excel"
	uniqueID := "test_" + generateUniqueID()
	_, errStr := processFile(invalidFilePath, fieldMappings, order, outputFormat, uniqueID)

	if errStr == "" || !strings.Contains(errStr, "Error opening file") {
		t.Errorf("expected error string for invalid file path: got %v", errStr)
	}
}

func TestProcessFileCSVOutput(t *testing.T) {
	tempFile, err := os.CreateTemp("./uploads", "test_process_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	fileContent := `Account Number,Account Active,Customer Name,Customer ID
	1234,Yes,John Doe,1001
	2345,No,Jane Smith,1002`
	_, err = tempFile.WriteString(fileContent)
	if err != nil {
		t.Fatal(err)
	}

	fieldMappings := map[string]string{
		"Client Code":    "Account Number",
		"Customer ID":    "Customer ID",
		"Account Number": "Account Number",
	}
	order := []string{"Client Code", "Customer ID", "Account Number"}
	outputFormat := "csv"
	uniqueID := "test_" + generateUniqueID()

	summary, processedFilePath := processFile(tempFile.Name(), fieldMappings, order, outputFormat, uniqueID)

	if summary == "" {
		t.Errorf("unexpected empty summary")
	}

	if processedFilePath == "" || !strings.HasSuffix(processedFilePath, ".csv") {
		t.Errorf("expected a valid processed CSV file path, got %v", processedFilePath)
	}
}

func TestGetFieldConfig(t *testing.T) {
	testConfigDir, err := os.MkdirTemp("", "test_config_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testConfigDir)

	originalConfigFile := "config/field_config.json"
	tempConfigFile := filepath.Join(testConfigDir, "field_config.json")

	tempConfig := `{
        "fields": [
            {
                "name": "Client_Code",
                "displayName": "Client Code",
                "isMandatory": true
            },
            {
                "name": "Customer_ID",
                "displayName": "Customer ID",
                "isMandatory": true
            }
        ]
    }`

	err = os.WriteFile(tempConfigFile, []byte(tempConfig), 0644)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(originalConfigFile); err == nil {
		backupFile := originalConfigFile + ".backup"
		if err := os.Rename(originalConfigFile, backupFile); err != nil {
			t.Fatal(err)
		}
		defer func() {
			os.Remove(originalConfigFile)
			os.Rename(backupFile, originalConfigFile)
		}()
	}

	err = os.MkdirAll(filepath.Dir(originalConfigFile), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(tempConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(originalConfigFile, input, 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = InitConfig()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/config", nil)
	recorder := httptest.NewRecorder()
	http.HandlerFunc(getFieldConfig).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expectedContentType := "application/json"
	if contentType := recorder.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	if !strings.Contains(recorder.Body.String(), "Client Code") {
		t.Errorf("response missing expected field 'Client Code': got %v", recorder.Body.String())
	}
}

func TestConfigInitialization(t *testing.T) {
	testConfigDir, err := os.MkdirTemp("", "test_config_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testConfigDir)

	originalConfigFile := "config/field_config.json"

	validConfig := `{
        "fields": [
            {
                "name": "Client_Code",
                "displayName": "Client Code",
                "isMandatory": true
            },
            {
                "name": "Customer_ID",
                "displayName": "Customer ID",
                "isMandatory": false
            }
        ]
    }`

	if _, err := os.Stat(originalConfigFile); err == nil {
		backupFile := originalConfigFile + ".backup"
		if err := os.Rename(originalConfigFile, backupFile); err != nil {
			t.Fatal(err)
		}
		defer func() {
			os.Remove(originalConfigFile)
			os.Rename(backupFile, originalConfigFile)
		}()
	}

	err = os.MkdirAll(filepath.Dir(originalConfigFile), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(originalConfigFile, []byte(validConfig), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = InitConfig()
	if err != nil {
		t.Errorf("failed to initialize valid config: %v", err)
	}

	invalidConfig := `{
        "fields": [
            {
                "name": "Client_Code",
                "displayName": "Client Code",
                "isMandatory": true,
            } // invalid JSON - extra comma
        ]
    }`

	err = os.WriteFile(originalConfigFile, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = InitConfig()
	if err == nil {
		t.Error("expected error with invalid JSON config, got nil")
	}
}

func TestGenerateMarkdownTable(t *testing.T) {
	headers := []string{"Name", "Age", "City"}
	rows := [][]string{
		{"John Doe", "30", "New York"},
		{"Jane Smith", "25", "Los Angeles"},
		{"Bob | Johnson", "35", "Chicago"}, // Test pipe character escaping
	}

	result := generateMarkdownTable(headers, rows)

	expected := "| Name | Age | City | \n| --- | --- | --- |\n| John Doe | 30 | New York | \n| Jane Smith | 25 | Los Angeles | \n| Bob \\| Johnson | 35 | Chicago | \n"

	if result != expected {
		t.Errorf("Markdown table generation failed.\nExpected (%v):\n%s\nGot (%v):\n%s",
			[]byte(expected), expected, []byte(result), result)
	}
}

func TestProcessFileMarkdownOutput(t *testing.T) {
	tempFile, err := os.CreateTemp("./uploads", "test_process_*.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	excelFile := excelize.NewFile()
	sheetName := "Sheet1"
	excelFile.SetSheetName("Sheet1", sheetName)

	headers := []string{"Account Number", "Account Active", "Customer Name"}
	data := [][]string{
		{"1234", "Yes", "John Doe"},
		{"5678", "No", "Jane Smith"},
	}

	for i, header := range headers {
		cell := string(rune('A'+i)) + "1"
		excelFile.SetCellValue(sheetName, cell, header)
	}

	for rowIndex, row := range data {
		for colIndex, value := range row {
			cell := string(rune('A'+colIndex)) + string(rune('2'+rowIndex))
			excelFile.SetCellValue(sheetName, cell, value)
		}
	}

	if err := excelFile.SaveAs(tempFile.Name()); err != nil {
		t.Fatal(err)
	}

	fieldMappings := map[string]string{
		"Account Number": "Account Number",
		"Account Active": "Account Active",
		"Customer Name":  "Customer Name",
	}
	order := []string{"Account Number", "Account Active", "Customer Name"}
	uniqueID := "test_" + generateUniqueID()

	summary, outputPath := processFile(tempFile.Name(), fieldMappings, order, "markdown", uniqueID)

	if !strings.Contains(summary, "Total Rows Processed") {
		t.Error("Summary missing expected content")
	}

	if !strings.HasSuffix(outputPath, ".md") {
		t.Error("Expected markdown file output")
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal("Failed to read output file")
	}

	markdownContent := string(content)
	if !strings.Contains(markdownContent, "# Data Processing Report") {
		t.Error("Markdown output missing expected header")
	}
	if !strings.Contains(markdownContent, "| Account Number |") {
		t.Error("Markdown output missing expected table header")
	}
}

func TestHandleAPIConfig(t *testing.T) {
	// Initialize config
	if err := InitConfig(); err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// Initialize API keys
	auth.InitAPIKeys()

	testCases := []struct {
		name          string
		apiKey        string
		expectedCode  int
		expectedError string
	}{
		{
			name:         "Valid API Key",
			apiKey:       "test-api-key-1",
			expectedCode: http.StatusOK,
		},
		{
			name:          "Missing API Key",
			apiKey:        "",
			expectedCode:  http.StatusUnauthorized,
			expectedError: "API key is missing",
		},
		{
			name:          "Invalid API Key",
			apiKey:        "invalid-key",
			expectedCode:  http.StatusUnauthorized,
			expectedError: "Invalid API key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest("GET", "/api/v1/config", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Add API key if present
			if tc.apiKey != "" {
				req.Header.Set("X-API-Key", tc.apiKey)
			}

			// Create a ResponseRecorder
			rr := httptest.NewRecorder()
			handler := auth.RequireAPIKey(handleAPIConfig)

			// Call the handler
			handler.ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != tc.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedCode)
			}

			// For error cases, check the error message
			if tc.expectedError != "" {
				if !strings.Contains(rr.Body.String(), tc.expectedError) {
					t.Errorf("handler returned unexpected error: got %v want %v", rr.Body.String(), tc.expectedError)
				}
			}

			// For success case, verify response content
			if tc.expectedCode == http.StatusOK {
				var response FieldConfigResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				// Verify mandatory fields exist
				if len(response.MandatoryFields) == 0 {
					t.Error("Expected mandatory fields in response")
				}

				// Verify fields array exists
				if len(response.Fields) == 0 {
					t.Error("Expected fields in response")
				}
			}
		})
	}
}

func TestHandleAPIProcess(t *testing.T) {
	// Initialize config and API keys
	if err := InitConfig(); err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}
	auth.InitAPIKeys()

	// Create a test file
	fileContent := `Account Number,Account Active,Customer Name
1234,Yes,John Doe
5678,No,Jane Smith`

	tempFile, err := os.CreateTemp("./uploads", "test_upload_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(fileContent); err != nil {
		t.Fatal(err)
	}
	if _, err := tempFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name          string
		apiKey        string
		expectedCode  int
		expectedError string
	}{
		{
			name:         "Valid API Key",
			apiKey:       "test-api-key-1",
			expectedCode: http.StatusOK,
		},
		{
			name:          "Missing API Key",
			apiKey:        "",
			expectedCode:  http.StatusUnauthorized,
			expectedError: "API key is missing",
		},
		{
			name:          "Invalid API Key",
			apiKey:        "invalid-key",
			expectedCode:  http.StatusUnauthorized,
			expectedError: "Invalid API key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new multipart form
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			// Add the file
			file, err := os.Open(tempFile.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			part, err := writer.CreateFormFile("file", filepath.Base(tempFile.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.Copy(part, file); err != nil {
				t.Fatal(err)
			}

			// Add the mappings
			mappings := map[string]string{
				"Account_Number": "Account Number",
				"Account_Active": "Account Active",
				"Customer_Name":  "Customer Name",
			}
			mappingsJSON, err := json.Marshal(mappings)
			if err != nil {
				t.Fatal(err)
			}
			if err := writer.WriteField("mappings", string(mappingsJSON)); err != nil {
				t.Fatal(err)
			}

			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}

			// Create the request
			req := httptest.NewRequest("POST", "/api/v1/process", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Add API key if present
			if tc.apiKey != "" {
				req.Header.Set("X-API-Key", tc.apiKey)
			}

			// Create a ResponseRecorder
			rr := httptest.NewRecorder()
			handler := auth.RequireAPIKey(handleAPIProcess)

			// Call the handler
			handler.ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != tc.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedCode)
			}

			// For error cases, check the error message
			if tc.expectedError != "" {
				if !strings.Contains(rr.Body.String(), tc.expectedError) {
					t.Errorf("handler returned unexpected error: got %v want %v", rr.Body.String(), tc.expectedError)
				}
			}

			// For success case, verify response headers
			if tc.expectedCode == http.StatusOK {
				if contentType := rr.Header().Get("Content-Type"); contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
					t.Errorf("handler returned wrong content type: got %v", contentType)
				}

				if disposition := rr.Header().Get("Content-Disposition"); disposition == "" {
					t.Error("Expected Content-Disposition header")
				}

				if summary := rr.Header().Get("X-Processing-Summary"); summary == "" {
					t.Error("Expected X-Processing-Summary header")
				}
			}
		})
	}
}

func TestUIRoutesWithAPIKey(t *testing.T) {
	// UI routes should work with or without API key
	routes := []string{"/", "/upload", "/config"}
	apiKey := "test-api-key-1"

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			// Test with API key
			req := httptest.NewRequest("GET", route, nil)
			req.Header.Set("X-API-Key", apiKey)
			rr := httptest.NewRecorder()
			http.DefaultServeMux.ServeHTTP(rr, req)

			if status := rr.Code; status == http.StatusUnauthorized {
				t.Errorf("UI route %s failed with API key: got status %v", route, status)
			}

			// Test without API key
			req = httptest.NewRequest("GET", route, nil)
			rr = httptest.NewRecorder()
			http.DefaultServeMux.ServeHTTP(rr, req)

			if status := rr.Code; status == http.StatusUnauthorized {
				t.Errorf("UI route %s failed without API key: got status %v", route, status)
			}
		})
	}
}

func TestHandleAPIProcessInvalidMethod(t *testing.T) {
	// Initialize API keys
	auth.InitAPIKeys()

	req := httptest.NewRequest("GET", "/api/v1/process", nil)
	req.Header.Set("X-API-Key", "test-api-key-1")
	rr := httptest.NewRecorder()
	handler := auth.RequireAPIKey(handleAPIProcess)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler allowed wrong HTTP method: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHandleAPIProcessMalformedJSON(t *testing.T) {
	// Initialize API keys
	auth.InitAPIKeys()

	// Create a test file
	fileContent := "Account Number,Account Active\n1234,Yes"
	tempFile, err := os.CreateTemp("./uploads", "test_upload_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(fileContent); err != nil {
		t.Fatal(err)
	}

	// Create a multipart form with malformed JSON
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add the file
	file, err := os.Open(tempFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(tempFile.Name()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, file); err != nil {
		t.Fatal(err)
	}

	// Add malformed JSON mappings
	malformedJSON := `{"key": "value", }` // Invalid JSON
	if err := writer.WriteField("mappings", malformedJSON); err != nil {
		t.Fatal(err)
	}

	writer.Close()

	// Create and send request
	req := httptest.NewRequest("POST", "/api/v1/process", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", "test-api-key-1")

	rr := httptest.NewRecorder()
	handler := auth.RequireAPIKey(handleAPIProcess)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler didn't reject malformed JSON: got %v want %v", status, http.StatusBadRequest)
	}

	if !strings.Contains(rr.Body.String(), "Invalid field mappings format") {
		t.Errorf("handler didn't return expected error message: got %v", rr.Body.String())
	}
}

func TestHandleAPIProcessEmptyFile(t *testing.T) {
	// Initialize API keys
	auth.InitAPIKeys()

	// Create an empty file
	tempFile, err := os.CreateTemp("./uploads", "test_upload_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Create a multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add the empty file
	file, err := os.Open(tempFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(tempFile.Name()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, file); err != nil {
		t.Fatal(err)
	}

	// Add valid mappings
	mappings := map[string]string{
		"Account_Number": "Account Number",
	}
	mappingsJSON, err := json.Marshal(mappings)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("mappings", string(mappingsJSON)); err != nil {
		t.Fatal(err)
	}

	writer.Close()

	// Create and send request
	req := httptest.NewRequest("POST", "/api/v1/process", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", "test-api-key-1")

	rr := httptest.NewRecorder()
	handler := auth.RequireAPIKey(handleAPIProcess)
	handler.ServeHTTP(rr, req)

	// The exact response code might depend on your implementation
	// but it should indicate an error condition
	if status := rr.Code; status == http.StatusOK {
		t.Error("handler accepted empty file without error")
	}
}

func TestHandleAPIProcessDifferentOutputFormats(t *testing.T) {
	// Initialize config and API keys
	if err := InitConfig(); err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}
	auth.InitAPIKeys()

	// Create a test file
	fileContent := `Account Number,Account Active,Customer Name
1234,Yes,John Doe
5678,No,Jane Smith`

	tempFile, err := os.CreateTemp("./uploads", "test_upload_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(fileContent); err != nil {
		t.Fatal(err)
	}

	outputFormats := []struct {
		format      string
		contentType string
	}{
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"csv", "text/csv"},
		{"markdown", "text/markdown"},
	}

	for _, of := range outputFormats {
		t.Run(of.format, func(t *testing.T) {
			// Create a multipart form
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			// Add the file
			file, err := os.Open(tempFile.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			part, err := writer.CreateFormFile("file", filepath.Base(tempFile.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.Copy(part, file); err != nil {
				t.Fatal(err)
			}

			// Add mappings
			mappings := map[string]string{
				"Account_Number": "Account Number",
				"Account_Active": "Account Active",
				"Customer_Name":  "Customer Name",
			}
			mappingsJSON, err := json.Marshal(mappings)
			if err != nil {
				t.Fatal(err)
			}
			if err := writer.WriteField("mappings", string(mappingsJSON)); err != nil {
				t.Fatal(err)
			}

			// Add output format
			if err := writer.WriteField("outputFormat", of.format); err != nil {
				t.Fatal(err)
			}

			writer.Close()

			// Create and send request
			req := httptest.NewRequest("POST", "/api/v1/process", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("X-API-Key", "test-api-key-1")

			rr := httptest.NewRecorder()
			handler := auth.RequireAPIKey(handleAPIProcess)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler failed for format %s: got status %v", of.format, status)
			}

			if contentType := rr.Header().Get("Content-Type"); contentType != of.contentType {
				t.Errorf("wrong content type for format %s: got %v want %v", of.format, contentType, of.contentType)
			}

			if disposition := rr.Header().Get("Content-Disposition"); disposition == "" {
				t.Error("Expected Content-Disposition header")
			}

			if summary := rr.Header().Get("X-Processing-Summary"); summary == "" {
				t.Error("Expected X-Processing-Summary header")
			}
		})
	}
}

// TestUploadDownloadWorkflow tests the complete end-to-end flow:
// Upload file -> Get response with filename -> Verify file exists -> Download file -> Verify content
// This test would have caught the bug where downloads returned 404 due to filename mismatch
func TestUploadDownloadWorkflow(t *testing.T) {
	// Test data with both successful and missing data rows
	fileContent := `Account Number,Account Active,Customer Name,Customer ID
1234,Yes,John Doe,1001
2345,No,Jane Smith,1002
3456,Yes,Bob Johnson,1003`

	outputFormats := []struct {
		format              string
		expectedExtension   string
		hasMissingDataFile  bool
		missingDataExtension string
	}{
		{"excel", ".xlsx", false, ""},
		{"csv", ".csv", true, ".csv"},
		{"markdown", ".md", true, ".md"},
	}

	for _, of := range outputFormats {
		t.Run(of.format, func(t *testing.T) {
			// Step 1: Create and upload a test file
			tempFile, err := os.CreateTemp("./uploads", "test_e2e_*.csv")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tempFile.Name())

			_, err = tempFile.WriteString(fileContent)
			if err != nil {
				t.Fatal(err)
			}
			_, err = tempFile.Seek(0, 0)
			if err != nil {
				t.Fatal(err)
			}

			// Create multipart form
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			part, err := writer.CreateFormFile("fileInput", filepath.Base(tempFile.Name()))
			if err != nil {
				t.Fatal(err)
			}
			_, err = io.Copy(part, tempFile)
			if err != nil {
				t.Fatal(err)
			}

			// Add field mappings
			_ = writer.WriteField("mapping_Client_Code", "Account Number")
			_ = writer.WriteField("mapping_Customer_ID", "Customer ID")
			_ = writer.WriteField("mapping_Account_ID", "Account Number")
			_ = writer.WriteField("outputFormat", of.format)

			writer.Close()

			// Step 2: Upload via handleUpload
			uploadReq := httptest.NewRequest("POST", "/upload", &body)
			uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

			uploadRecorder := httptest.NewRecorder()
			http.HandlerFunc(handleUpload).ServeHTTP(uploadRecorder, uploadReq)

			if status := uploadRecorder.Code; status != http.StatusOK {
				t.Fatalf("Upload failed with status %v: %s", status, uploadRecorder.Body.String())
			}

			// Step 3: Parse JSON response and extract filenames
			var response map[string]interface{}
			if err := json.Unmarshal(uploadRecorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse upload response: %v", err)
			}

			outputFilename, ok := response["outputFilename"].(string)
			if !ok || outputFilename == "" {
				t.Fatalf("No outputFilename in response: %+v", response)
			}

			// Verify filename has expected extension
			if !strings.HasSuffix(outputFilename, of.expectedExtension) {
				t.Errorf("Expected filename to end with %s, got %s", of.expectedExtension, outputFilename)
			}

			// Verify filename contains unique ID format (timestamp_random_*)
			if !strings.Contains(outputFilename, "_") {
				t.Errorf("Expected filename to contain unique ID with underscore, got %s", outputFilename)
			}

			// Step 4: Verify the file actually exists on disk
			outputPath := filepath.Join("./uploads", outputFilename)
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatalf("Output file does not exist at %s", outputPath)
			}
			defer os.Remove(outputPath) // Clean up

			// Step 5: Download the file via handleDownload
			downloadReq := httptest.NewRequest("GET", "/download?file="+outputFilename, nil)
			downloadRecorder := httptest.NewRecorder()
			http.HandlerFunc(handleDownload).ServeHTTP(downloadRecorder, downloadReq)

			if status := downloadRecorder.Code; status != http.StatusOK {
				t.Fatalf("Download failed with status %v for file %s", status, outputFilename)
			}

			// Step 6: Verify downloaded file is not empty
			downloadedContent := downloadRecorder.Body.Bytes()
			if len(downloadedContent) == 0 {
				t.Fatal("Downloaded file is empty")
			}

			// Step 7: For CSV and Markdown, verify missing data file exists and is downloadable
			if of.hasMissingDataFile {
				missingFilename, ok := response["missingFilename"].(string)
				if !ok || missingFilename == "" {
					t.Errorf("Expected missingFilename in response for format %s", of.format)
				} else {
					// Verify missing data file exists
					missingPath := filepath.Join("./uploads", missingFilename)
					if _, err := os.Stat(missingPath); os.IsNotExist(err) {
						t.Errorf("Missing data file does not exist at %s", missingPath)
					} else {
						defer os.Remove(missingPath) // Clean up

						// Verify missing data file is downloadable
						missingDownloadReq := httptest.NewRequest("GET", "/download?file="+missingFilename, nil)
						missingDownloadRecorder := httptest.NewRecorder()
						http.HandlerFunc(handleDownload).ServeHTTP(missingDownloadRecorder, missingDownloadReq)

						if status := missingDownloadRecorder.Code; status != http.StatusOK {
							t.Errorf("Missing data download failed with status %v for file %s", status, missingFilename)
						}

						if len(missingDownloadRecorder.Body.Bytes()) == 0 {
							t.Error("Downloaded missing data file is empty")
						}
					}
				}
			}

			// Step 8: Verify content quality for CSV format (easier to validate)
			if of.format == "csv" {
				csvContent := string(downloadedContent)
				// Should have headers
				if !strings.Contains(csvContent, "Client_Code") {
					t.Error("CSV missing expected header 'Client_Code'")
				}
				// Should use pipe delimiter
				if !strings.Contains(csvContent, "|") {
					t.Error("CSV should use pipe delimiter")
				}
			}

			t.Logf("✅ End-to-end test passed for format %s: uploaded -> response verified -> file exists -> downloaded successfully", of.format)
		})
	}
}
