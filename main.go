package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	http.HandleFunc("/", serveUI)
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/download", handleDownload)
	http.ListenAndServe(":8080", nil)
}

func serveUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("field_mapping_ui.html")
	if err != nil {
		http.Error(w, "Could not load UI", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data to handle file upload and field mappings
	err := r.ParseMultipartForm(10 << 20) // limit upload size to 10MB
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("fileInput")
	if err != nil {
		http.Error(w, "No file uploaded. Please choose a file to upload.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file type
	if !strings.HasSuffix(handler.Filename, ".xlsx") && !strings.HasSuffix(handler.Filename, ".csv") {
		http.Error(w, "Invalid file type. Only .csv and .xlsx files are allowed", http.StatusBadRequest)
		return
	}

	// Save the uploaded file temporarily
	tempDir := "./uploads"
	os.MkdirAll(tempDir, os.ModePerm)
	tempFilePath := filepath.Join(tempDir, handler.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = tempFile.ReadFrom(file)
	if err != nil {
		http.Error(w, "Unable to save file content", http.StatusInternalServerError)
		return
	}

	// Extract field mappings from form
	fieldMappings := make(map[string]string)
	for key, values := range r.PostForm {
		if strings.HasPrefix(key, "mapping_") {
			expectedField := strings.TrimPrefix(key, "mapping_")
			fieldMappings[expectedField] = values[0]
		}
	}

	// Process the uploaded file using the field mappings
	summary, outputFilePath := processFile(tempFilePath, fieldMappings)

	fmt.Fprintf(w, "File uploaded successfully and mappings are: %+v\n\nSummary Report:\n%s\n\nDownload Processed Data: <a href=\"/download?file=%s\">Download Excel</a>", fieldMappings, summary, outputFilePath)
}

func processFile(filePath string, fieldMappings map[string]string) (string, string) {
	var rows [][]string
	var err error

	// Determine the file type
	if strings.HasSuffix(filePath, ".xlsx") {
		// Process as .xlsx file
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return "Error opening file.", "Error opening file."
		}

		sheetName := f.GetSheetName(0)
		rows, err = f.GetRows(sheetName)
		if err != nil {
			fmt.Println("Error reading sheet rows:", err)
			return "Error reading sheet rows.", "Error reading sheet rows."
		}
	} else if strings.HasSuffix(filePath, ".csv") {
		// Process as .csv file
		csvFile, err := os.Open(filePath)
		if err != nil {
			fmt.Println("Error opening CSV file:", err)
			return "Error opening CSV file.", "Error opening CSV file."
		}
		defer csvFile.Close()

		reader := csv.NewReader(csvFile)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println("Error reading CSV file:", err)
				return "Error reading CSV file.", "Error reading CSV file."
			}
			rows = append(rows, record)
		}
	} else {
		return "Unsupported file format.", "Unsupported file format."
	}

	if len(rows) == 0 {
		return "No data found in the file.", "No data found in the file."
	}

	// Proceed with processing the rows (common for both .xlsx and .csv)
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString("Data Mapping Summary:\n")
	missingCount := 0
	successfulRows := 0

	// Normalize headers in the first row
	normalizedHeaders := make([]string, len(rows[0]))
	for i, header := range rows[0] {
		normalizedHeaders[i] = strings.TrimSpace(strings.ToLower(header))
	}

	// Create a new file for successful rows and missing rows
	outputFile := excelize.NewFile()
	outputFile.NewSheet("ProcessedData")
	outputFile.NewSheet("MissingData")
	outputFile.DeleteSheet("Sheet1")

	// Set headers for ProcessedData and MissingData sheets
	processedHeaders := []string{}
	for expectedField := range fieldMappings {
		processedHeaders = append(processedHeaders, expectedField)
	}
	outputFile.SetSheetRow("ProcessedData", "A1", &processedHeaders)
	outputFile.SetSheetRow("MissingData", "A1", &processedHeaders)

	outputRowIndex := 2
	missingRowIndex := 2

	// Process rows based on the field mappings
	for i, row := range rows {
		// Skip header row
		if i == 0 {
			continue
		}

		processedRow := make([]string, len(fieldMappings))
		missingRow := make([]string, len(fieldMappings))
		rowMissingFields := []string{}
		rowSuccess := true
		for fieldIndex, expectedField := range processedHeaders {
			// Normalize column header for comparison
			normalizedColumnHeader := strings.TrimSpace(strings.ToLower(fieldMappings[expectedField]))

			// Find the column index for the current mapping
			columnIndex := -1
			for j, header := range normalizedHeaders {
				if header == normalizedColumnHeader {
					columnIndex = j
					break
				}
			}

			if columnIndex != -1 && columnIndex < len(row) && strings.TrimSpace(row[columnIndex]) != "" {
				processedRow[fieldIndex] = row[columnIndex]
				missingRow[fieldIndex] = row[columnIndex]
			} else {
				rowMissingFields = append(rowMissingFields, expectedField)
				missingRow[fieldIndex] = "MISSING"
				rowSuccess = false
			}
		}

		if rowSuccess {
			successfulRows++
			outputFile.SetSheetRow("ProcessedData", fmt.Sprintf("A%d", outputRowIndex), &processedRow)
			outputRowIndex++
		} else {
			missingCount++
			outputFile.SetSheetRow("MissingData", fmt.Sprintf("A%d", missingRowIndex), &missingRow)
			missingRowIndex++
			summaryBuilder.WriteString(fmt.Sprintf("Row %d: Missing fields - %s\n", i+1, strings.Join(rowMissingFields, ", ")))
		}
	}

	summaryBuilder.WriteString(fmt.Sprintf("\nTotal Rows Processed: %d\n", len(rows)-1))
	summaryBuilder.WriteString(fmt.Sprintf("Successful Rows: %d\n", successfulRows))
	summaryBuilder.WriteString(fmt.Sprintf("Rows with Missing Data: %d\n", missingCount))

	// Output summary to the console as well
	fmt.Println(summaryBuilder.String())

	// Save the output file
	outputFilePath := "./uploads/processed_data.xlsx"
	err = outputFile.SaveAs(outputFilePath)
	if err != nil {
		fmt.Println("Error saving output file:", err)
		return summaryBuilder.String(), ""
	}

	return summaryBuilder.String(), outputFilePath
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	filePath := "./uploads/" + file
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}
