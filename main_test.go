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

	"github.com/xuri/excelize/v2"
)

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

	if !strings.Contains(recorder.Body.String(), "File uploaded successfully") {
		t.Errorf("handler returned unexpected body: got %v", recorder.Body.String())
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
	summary, errStr := processFile(tempFile.Name(), fieldMappings, order, outputFormat)

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
	_, errStr := processFile(invalidFilePath, fieldMappings, order, outputFormat)

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

	summary, processedFilePath := processFile(tempFile.Name(), fieldMappings, order, outputFormat)

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

	summary, outputPath := processFile(tempFile.Name(), fieldMappings, order, "markdown")

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

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/api/v1/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleAPIConfig)

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body contains expected fields
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

func TestHandleAPIProcess(t *testing.T) {
	// Initialize config
	if err := InitConfig(); err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// Create a buffer to write our multipart form to
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add the file to the form
	file, err := os.Open("uploads/synthetic_test_data.xlsx")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()

	fw, err := w.CreateFormFile("file", "synthetic_test_data.xlsx")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := io.Copy(fw, file); err != nil {
		t.Fatalf("Failed to copy file content: %v", err)
	}

	// Add the mappings to the form
	mappings := map[string]string{
		"Client_Code": "Client Code",
		"Customer_ID": "Customer ID",
		"Account_ID":  "Account Number",
	}
	mappingsJSON, err := json.Marshal(mappings)
	if err != nil {
		t.Fatalf("Failed to marshal mappings: %v", err)
	}
	if err := w.WriteField("mappings", string(mappingsJSON)); err != nil {
		t.Fatalf("Failed to write mappings field: %v", err)
	}

	// Add the output format to the form
	if err := w.WriteField("outputFormat", "xlsx"); err != nil {
		t.Fatalf("Failed to write output format field: %v", err)
	}

	// Close the writer
	w.Close()

	// Create a request with the form
	req, err := http.NewRequest("POST", "/api/v1/process", &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleAPIProcess)

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response headers
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("handler returned wrong content type: got %v", contentType)
	}

	if disposition := rr.Header().Get("Content-Disposition"); disposition == "" {
		t.Error("Expected Content-Disposition header")
	}

	if summary := rr.Header().Get("X-Processing-Summary"); summary == "" {
		t.Error("Expected X-Processing-Summary header")
	}

	// Check that we got some file content
	if len(rr.Body.Bytes()) == 0 {
		t.Error("Expected file content in response")
	}
}
