package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"import/auth"
	"import/config"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "import/docs" // swagger docs

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/xuri/excelize/v2"
)

var fieldConfig *config.FieldConfig

// @title           Field Mapping API
// @version         1.0
// @description     API for processing and mapping fields in CSV and XLSX files.

// @contact.name   Github
// @contact.url    https://github.com/ShaunBoughey/excel-mapper

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description API key authentication required for all API endpoints

// @accept multipart/form-data
// @produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @produce text/csv
// @produce text/markdown

func InitConfig() error {
	configFile, err := os.ReadFile("config/field_config.json")
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	fieldConfig = &config.FieldConfig{}
	if err := json.Unmarshal(configFile, fieldConfig); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}
	return nil
}

func init() {
	// Call InitConfig in init, but handle the error appropriately for production
	if err := InitConfig(); err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Initialize API keys
	auth.InitAPIKeys()
}

func main() {
	// Serve static UI files (CSS, JS)
	uiFS := http.FileServer(http.Dir("ui"))
	http.Handle("/ui/", http.StripPrefix("/ui/", uiFS))

	// UI routes
	http.HandleFunc("/", serveUI)
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/download", handleDownload)
	http.HandleFunc("/config", getFieldConfig)

	// API routes with authentication
	http.HandleFunc("/api/v1/config", auth.RequireAPIKey(handleAPIConfig))
	http.HandleFunc("/api/v1/process", auth.RequireAPIKey(handleAPIProcess))

	// Serve swagger files
	fs := http.FileServer(http.Dir("docs"))
	http.Handle("/docs/", http.StripPrefix("/docs/", fs))

	// Swagger UI handler
	http.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	log.Printf("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func serveUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/index.html")
	if err != nil {
		http.Error(w, "Could not load UI", http.StatusInternalServerError)
		return
	}

	// Add base URL to template data
	data := map[string]interface{}{
		"BaseURL": "http://" + r.Host,
	}

	tmpl.Execute(w, data)
}

func getFieldConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fields":          fieldConfig.Fields,
		"mandatoryFields": fieldConfig.GetMandatoryFields(),
	})
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
	order := fieldConfig.GetOrderedFields()

	// For multipart forms, use MultipartForm.Value instead of PostForm
	formValues := r.MultipartForm.Value
	for key, values := range formValues {
		if strings.HasPrefix(key, "mapping_") {
			expectedField := strings.TrimPrefix(key, "mapping_")
			if len(values) > 0 && values[0] != "" {
				fieldMappings[expectedField] = values[0]
			}
			if !contains(order, expectedField) {
				order = append(order, expectedField)
			}
		}
	}

	// Get output format from multipart form
	outputFormat := "excel"
	if formats, ok := formValues["outputFormat"]; ok && len(formats) > 0 {
		outputFormat = formats[0]
	}

	// Process the uploaded file using the field mappings
	summary, _ := processFile(tempFilePath, fieldMappings, order, outputFormat)

	fmt.Fprintf(w, "File uploaded successfully and mappings are: %+v\n\nSummary Report:\n%s\n", fieldMappings, summary)
}

// readInputFile reads and parses the input file based on its extension
func readInputFile(filePath string) ([][]string, error) {
	if strings.HasSuffix(filePath, ".xlsx") {
		return readXLSXFile(filePath)
	} else if strings.HasSuffix(filePath, ".csv") {
		return readCSVFile(filePath)
	}
	return nil, fmt.Errorf("unsupported file format")
}

func readXLSXFile(filePath string) ([][]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening xlsx file: %v", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("error reading sheet rows: %v", err)
	}
	return rows, nil
}

func readCSVFile(filePath string) ([][]string, error) {
	csvFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %v", err)
	}
	defer csvFile.Close()

	var rows [][]string
	reader := csv.NewReader(csvFile)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV file: %v", err)
		}
		rows = append(rows, record)
	}
	return rows, nil
}

// normalizeHeaders converts headers to lowercase and trims whitespace
func normalizeHeaders(headers []string) []string {
	normalized := make([]string, len(headers))
	for i, header := range headers {
		normalized[i] = strings.TrimSpace(strings.ToLower(header))
	}
	return normalized
}

// createOutputWorkbook creates a new Excel workbook with ProcessedData and MissingData sheets
func createOutputWorkbook(headers []string) *excelize.File {
	outputFile := excelize.NewFile()
	outputFile.NewSheet("ProcessedData")
	outputFile.NewSheet("MissingData")
	outputFile.DeleteSheet("Sheet1")
	outputFile.SetSheetRow("ProcessedData", "A1", &headers)
	outputFile.SetSheetRow("MissingData", "A1", &headers)
	return outputFile
}

// generateProcessingSummary creates a formatted summary of the processing results
func generateProcessingSummary(totalRows, successfulRows, missingCount int, missingDetails string) string {
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString("Data Mapping Summary:\n")
	if missingDetails != "" {
		summaryBuilder.WriteString(missingDetails)
	}
	summaryBuilder.WriteString(fmt.Sprintf("\nTotal Rows Processed: %d\n", totalRows))
	summaryBuilder.WriteString(fmt.Sprintf("Successful Rows: %d\n", successfulRows))
	summaryBuilder.WriteString(fmt.Sprintf("Rows with Missing Data: %d\n", missingCount))
	return summaryBuilder.String()
}

// saveAsXLSX saves the output file as an Excel workbook
func saveAsXLSX(outputFile *excelize.File, outputPath string) (string, error) {
	if err := outputFile.SaveAs(outputPath); err != nil {
		return "", fmt.Errorf("error saving output file: %w", err)
	}
	return outputPath, nil
}

// saveAsMarkdown saves the output file as Markdown with a report format
func saveAsMarkdown(outputFile *excelize.File, order []string, outputRowCount, missingRowCount int, summary string) (string, error) {
	outputFilePath := "./uploads/processed_data.md"
	mdFile, err := os.Create(outputFilePath)
	if err != nil {
		return "", fmt.Errorf("error creating markdown file: %w", err)
	}
	defer mdFile.Close()

	var processedRows [][]string
	processedRows = append(processedRows, order) // Add headers
	for rowIndex := 2; rowIndex < outputRowCount; rowIndex++ {
		row := make([]string, len(order))
		for j := range row {
			cell, _ := outputFile.GetCellValue("ProcessedData", fmt.Sprintf("%s%d", string(rune('A'+j)), rowIndex))
			row[j] = cell
		}
		processedRows = append(processedRows, row)
	}

	markdownContent := generateMarkdownTable(order, processedRows[1:])

	// Add summary section to markdown
	fullContent := fmt.Sprintf("# Data Processing Report\n\n## Summary\n\n```\n%s\n```\n\n## Processed Data\n\n%s",
		summary, markdownContent)

	_, err = mdFile.WriteString(fullContent)
	if err != nil {
		return "", fmt.Errorf("error writing markdown content: %w", err)
	}

	// Save missing rows to separate markdown file
	missingFilePath := "./uploads/missing_data.md"
	missingMdFile, err := os.Create(missingFilePath)
	if err != nil {
		return outputFilePath, fmt.Errorf("error creating missing data markdown file: %w", err)
	}
	defer missingMdFile.Close()

	var missingRows [][]string
	missingRows = append(missingRows, order)
	for rowIndex := 2; rowIndex < missingRowCount; rowIndex++ {
		row := make([]string, len(order))
		for j := range row {
			cell, _ := outputFile.GetCellValue("MissingData", fmt.Sprintf("%s%d", string(rune('A'+j)), rowIndex))
			row[j] = cell
		}
		missingRows = append(missingRows, row)
	}

	missingMarkdownContent := generateMarkdownTable(order, missingRows[1:])
	missingFullContent := fmt.Sprintf("# Missing Data Report\n\n## Missing Records\n\n%s", missingMarkdownContent)

	_, err = missingMdFile.WriteString(missingFullContent)
	if err != nil {
		return outputFilePath, fmt.Errorf("error writing missing data markdown content: %w", err)
	}

	return outputFilePath, nil
}

// saveAsCSV saves the output file as CSV with pipe delimiter
func saveAsCSV(outputFile *excelize.File, order []string, outputRowCount, missingRowCount int) (string, error) {
	outputFilePath := "./uploads/processed_data.csv"
	csvFile, err := os.Create(outputFilePath)
	if err != nil {
		return "", fmt.Errorf("error creating CSV file: %w", err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	csvWriter.Comma = '|'
	csvWriter.Write(order)
	// Write processed rows
	for rowIndex := 2; rowIndex < outputRowCount; rowIndex++ {
		row := make([]string, len(order))
		for j := range row {
			cell, _ := outputFile.GetCellValue("ProcessedData", fmt.Sprintf("%s%d", string(rune('A'+j)), rowIndex))
			row[j] = cell
		}
		csvWriter.Write(row)
	}
	csvWriter.Flush()

	// Save missing rows to separate CSV
	missingFilePath := "./uploads/missing_data.csv"
	missingCsvFile, err := os.Create(missingFilePath)
	if err != nil {
		return outputFilePath, fmt.Errorf("error creating missing data CSV file: %w", err)
	}
	defer missingCsvFile.Close()

	missingCsvWriter := csv.NewWriter(missingCsvFile)
	missingCsvWriter.Comma = '|'
	missingCsvWriter.Write(order)
	// Write missing rows
	for rowIndex := 2; rowIndex < missingRowCount; rowIndex++ {
		row := make([]string, len(order))
		for j := range row {
			cell, _ := outputFile.GetCellValue("MissingData", fmt.Sprintf("%s%d", string(rune('A'+j)), rowIndex))
			row[j] = cell
		}
		missingCsvWriter.Write(row)
	}
	missingCsvWriter.Flush()

	return outputFilePath, nil
}

// processRow processes a single row and returns the processed data, missing data, missing fields, and success status
func processRow(row []string, normalizedHeaders []string, fieldMappings map[string]string, order []string, fieldConfig *config.FieldConfig) (processedRow []string, missingRow []string, missingFields []string, isSuccess bool) {
	processedRow = make([]string, len(order))
	missingRow = make([]string, len(order))
	missingFields = make([]string, 0, len(order))
	isSuccess = true

	for fieldIndex, expectedField := range order {
		var isMandatory bool
		for _, field := range fieldConfig.Fields {
			if field.Name == expectedField {
				isMandatory = field.IsMandatory
				break
			}
		}

		mappedColumn := fieldMappings[expectedField]

		// If the mapping is empty (no column selected) and not mandatory,
		// just leave it blank without marking as MISSING
		if mappedColumn == "" && !isMandatory {
			processedRow[fieldIndex] = ""
			missingRow[fieldIndex] = ""
			continue
		}

		// Normalize column header for comparison
		normalizedColumnHeader := strings.TrimSpace(strings.ToLower(mappedColumn))

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
			// Only add to missing fields if it's mandatory
			if isMandatory {
				missingFields = append(missingFields, expectedField)
				isSuccess = false
				missingRow[fieldIndex] = "MISSING"
			} else {
				// For non-mandatory fields, only mark as MISSING if a mapping was selected
				if mappedColumn != "" {
					missingRow[fieldIndex] = "MISSING"
				} else {
					missingRow[fieldIndex] = ""
				}
			}
			processedRow[fieldIndex] = ""
		}
	}

	return processedRow, missingRow, missingFields, isSuccess
}

func processFile(filePath string, fieldMappings map[string]string, order []string, outputFormat string) (string, string) {
	rows, err := readInputFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error opening file: %v", err), "Error opening file"
	}

	if len(rows) == 0 {
		return "No data found in the file.", "No data found in the file"
	}

	// Proceed with processing the rows (common for both .xlsx and .csv)
	var missingDetailsBuilder strings.Builder
	missingCount := 0
	successfulRows := 0

	// Normalize headers in the first row
	normalizedHeaders := normalizeHeaders(rows[0])

	// Create a new file for successful rows and missing rows
	outputFile := createOutputWorkbook(order)

	outputRowIndex := 2
	missingRowIndex := 2

	// Process rows based on the field mappings
	for i, row := range rows {
		// Skip header row
		if i == 0 {
			continue
		}

		processedRow, missingRow, rowMissingFields, rowSuccess := processRow(row, normalizedHeaders, fieldMappings, order, fieldConfig)

		if rowSuccess {
			successfulRows++
			outputFile.SetSheetRow("ProcessedData", fmt.Sprintf("A%d", outputRowIndex), &processedRow)
			outputRowIndex++
		} else {
			missingCount++
			outputFile.SetSheetRow("MissingData", fmt.Sprintf("A%d", missingRowIndex), &missingRow)
			missingRowIndex++
			if len(rowMissingFields) > 0 {
				missingDetailsBuilder.WriteString(fmt.Sprintf("Row %d: Missing mandatory fields - %s\n", i+1, strings.Join(rowMissingFields, ", ")))
			}
		}
	}

	// Generate and output summary
	summary := generateProcessingSummary(len(rows)-1, successfulRows, missingCount, missingDetailsBuilder.String())
	fmt.Println(summary)

	// Save the output file based on user choice
	if outputFormat == "csv" {
		outputFilePath, err := saveAsCSV(outputFile, order, outputRowIndex, missingRowIndex)
		if err != nil {
			fmt.Println(err)
			return summary, ""
		}
		return summary, outputFilePath
	}

	if outputFormat == "markdown" {
		outputFilePath, err := saveAsMarkdown(outputFile, order, outputRowIndex, missingRowIndex, summary)
		if err != nil {
			fmt.Println(err)
			return summary, ""
		}
		return summary, outputFilePath
	}

	outputFilePath := "./uploads/processed_data.xlsx"
	outputFilePath, err = saveAsXLSX(outputFile, outputFilePath)
	if err != nil {
		fmt.Println(err)
		return summary, ""
	}

	return summary, outputFilePath
}

func generateMarkdownTable(headers []string, rows [][]string) string {
	var sb strings.Builder

	sb.WriteString("| ")
	for _, header := range headers {
		sb.WriteString(header + " | ")
	}
	sb.WriteString("\n|")

	for range headers {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")

	// Write data rows
	for _, row := range rows {
		sb.WriteString("| ")
		for _, cell := range row {
			escapedCell := strings.ReplaceAll(cell, "|", "\\|")
			sb.WriteString(escapedCell + " | ")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	if strings.Contains(file, "..") || strings.ContainsAny(file, `/\`) {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("./uploads", file)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type FieldConfigResponse struct {
	Fields []struct {
		Name        string `json:"name" example:"Client_Code"`
		DisplayName string `json:"displayName" example:"Client Code"`
		IsMandatory bool   `json:"isMandatory" example:"true"`
	} `json:"fields"`
	MandatoryFields []string `json:"mandatoryFields" example:"Client_Code,Customer_ID,Account_ID"`
}

// @Summary     Get field configuration
// @Description Get the configuration of all fields, including mandatory fields and field order
// @Tags        configuration
// @Accept      json
// @Produce     json
// @Security    ApiKeyAuth
// @Success     200 {object} FieldConfigResponse
// @Failure     401 {object} ErrorResponse "Unauthorized"
// @Failure     405 {object} ErrorResponse "Method Not Allowed"
// @Router      /config [get]
func handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fields":          fieldConfig.Fields,
		"mandatoryFields": fieldConfig.GetMandatoryFields(),
		"orderedFields":   fieldConfig.GetOrderedFields(),
	})
}

// ProcessResponse represents the file processing response
type ProcessResponse struct {
	Summary     string `json:"summary" example:"Total Rows Processed: 1000 Successful Rows: 1000 Rows with Missing Data: 0"`
	FileName    string `json:"fileName" example:"processed_data.xlsx"`
	ContentType string `json:"contentType" example:"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"`
}

// @Summary      Process file with field mappings
// @Description  Upload a file and process it according to provided field mappings
// @Tags         processing
// @Accept       multipart/form-data
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Produce      text/csv
// @Produce      text/markdown
// @Security     ApiKeyAuth
// @Param        file formData file true "File to process (CSV or XLSX)"
// @Param        mappings formData string true "JSON string of field mappings" example:"{\"Client_Code\":\"Client Code\",\"Customer_ID\":\"Customer ID\",\"Account_ID\":\"Account Number\"}"
// @Param        outputFormat formData string false "Output format" Enums(xlsx,csv,markdown) default(xlsx)
// @Success      200 {object} ProcessResponse
// @Header       200 {string} X-Processing-Summary "Total Rows Processed: 1000 Successful Rows: 1000 Rows with Missing Data: 0"
// @Header       200 {string} Content-Type "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
// @Header       200 {string} Content-Disposition "attachment; filename=\"processed_data.xlsx\""
// @Failure      400 {object} ErrorResponse "Bad Request"
// @Failure      401 {object} ErrorResponse "Unauthorized"
// @Failure      500 {object} ErrorResponse "Internal Server Error"
// @Router       /process [post]
func handleAPIProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get the file
	file, handler, err := r.FormFile("file")
	if err != nil {
		sendJSONError(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(handler.Filename, ".xlsx") && !strings.HasSuffix(handler.Filename, ".csv") {
		sendJSONError(w, "Invalid file type. Only .csv and .xlsx files are allowed", http.StatusBadRequest)
		return
	}

	// Get field mappings from JSON
	var fieldMappings map[string]string
	mappingsStr := r.FormValue("mappings")
	if err := json.Unmarshal([]byte(mappingsStr), &fieldMappings); err != nil {
		sendJSONError(w, "Invalid field mappings format", http.StatusBadRequest)
		return
	}

	// Save file temporarily
	tempDir := "./uploads"
	os.MkdirAll(tempDir, os.ModePerm)
	tempFilePath := filepath.Join(tempDir, handler.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		sendJSONError(w, "Unable to save file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = tempFile.ReadFrom(file)
	if err != nil {
		sendJSONError(w, "Unable to save file content", http.StatusInternalServerError)
		return
	}

	// Get output format
	outputFormat := r.FormValue("outputFormat")
	if outputFormat == "" {
		outputFormat = "xlsx" // Default format
	}

	// Process the file
	order := fieldConfig.GetOrderedFields()
	summary, outputPath := processFile(tempFilePath, fieldMappings, order, outputFormat)

	// Check if the output file exists
	if _, err := os.Stat(outputPath); err != nil {
		sendJSONError(w, "Failed to generate output file", http.StatusInternalServerError)
		return
	}

	// Read the file
	fileContent, err := os.ReadFile(outputPath)
	if err != nil {
		sendJSONError(w, "Failed to read output file", http.StatusInternalServerError)
		return
	}

	// Set appropriate headers based on output format
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if outputFormat == "csv" {
		contentType = "text/csv"
	} else if outputFormat == "markdown" {
		contentType = "text/markdown"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(outputPath)))
	w.Header().Set("X-Processing-Summary", summary)
	w.Write(fileContent)
}

func sendJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid field mappings format"`
}
