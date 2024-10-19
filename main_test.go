package main

import (
	"bytes"
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
	req, err := http.NewRequest("GET", "/download?file=./uploads/non_existent_file.xlsx", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	http.HandlerFunc(handleDownload).ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code for non-existent file: got %v want %v", status, http.StatusNotFound)
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

	summary, errStr := processFile(tempFile.Name(), fieldMappings)

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

	_, errStr := processFile(invalidFilePath, fieldMappings)

	if errStr == "" || !strings.Contains(errStr, "Error opening file") {
		t.Errorf("expected error string for invalid file path: got %v", errStr)
	}
}
