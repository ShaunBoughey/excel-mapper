![Coverage](https://img.shields.io/badge/Coverage-82.6%25-brightgreen)
[![CodeFactor](https://www.codefactor.io/repository/github/shaunboughey/excel-mapper/badge)](https://www.codefactor.io/repository/github/shaunboughey/excel-mapper)

# Excel Mapper

A high-performance service that allows you to map and transform fields between Excel (XLSX) and CSV files. It provides both a web UI and REST API interface for processing files.

## Features
- Support for both XLSX and CSV file formats
- Field mapping configuration
- Multiple output formats (XLSX, CSV, Markdown)
- REST API with Swagger documentation
- Web-based UI for interactive mapping
- Mandatory field validation
- Processing summary reports
- API key authentication for secure API access

## Getting Started

### Running the Service
1. Set up API keys (required for API access):
   ```bash
   # Windows PowerShell
   $env:API_KEYS="your-api-key-1,your-api-key-2"
   
   # Linux/Mac
   export API_KEYS="your-api-key-1,your-api-key-2"
   ```
2. Start the service:
   ```bash
   go run main.go
   ```
3. Access the web UI at: http://localhost:8080
4. Access the Swagger documentation at: http://localhost:8080/swagger/

### Using the Web UI
1. Open http://localhost:8080 in your browser
2. Upload your source file (XLSX or CSV)
3. Configure field mappings
4. Choose output format
5. Process the file and download results

### Using the REST API

#### Authentication
All API endpoints require an API key to be included in the request headers:
```
X-API-Key: your-api-key
```

#### 1. Get Field Configuration
```bash
curl -X GET http://localhost:8080/api/v1/config \
  -H "X-API-Key: your-api-key"
```

#### 2. Process File
```bash
curl -i -X POST http://localhost:8080/api/v1/process \
  -H "X-API-Key: your-api-key" \
  -F "file=@your_file.xlsx" \
  -F 'mappings={
    "Target_Field":"Source Field",
    "Another_Target":"Another Source"
  }' \
  -F "outputFormat=xlsx" \
  --output processed_data.xlsx
```

## API Documentation

### Authentication
All API endpoints require an API key to be passed in the `X-API-Key` header. API keys can be configured using the `API_KEYS` environment variable as a comma-separated list.

### GET /api/v1/config
Returns the field configuration including:
- Available fields
- Mandatory fields
- Field order

### POST /api/v1/process
Process a file with field mappings.

Parameters:
- `file`: The input file (XLSX or CSV)
- `mappings`: JSON string of field mappings
- `outputFormat`: Output format (xlsx, csv, markdown)

## Configuration
The service uses a configuration file at `config/field_config.json` to define:
- Available fields
- Mandatory fields
- Field display names
- Field order

## Technical Details

### Performance
- Handles files up to 10MB in size
- Efficient memory usage for large files
- Fast processing with Go's concurrent capabilities

### Security
- API key authentication for all API endpoints
- Input validation for all API endpoints
- File size limits
- Safe file handling
- No sensitive data exposure

## Troubleshooting

### Common Issues and Solutions

#### 1. Authentication Issues
- **Error**: "API key is missing"
  - **Cause**: No X-API-Key header in request
  - **Solution**: Add X-API-Key header with valid API key

- **Error**: "Invalid API key"
  - **Cause**: API key not recognized
  - **Solution**: Check API_KEYS environment variable is set correctly

#### 2. File Upload Issues
- **Error**: "File too large"
  - **Cause**: File exceeds 10MB limit
  - **Solution**: Split the file or increase limit in `main.go`

- **Error**: "Invalid file type"
  - **Cause**: File is not XLSX or CSV
  - **Solution**: Convert file to supported format

#### 3. Field Mapping Issues
- **Error**: "Missing mandatory field"
  - **Cause**: Required field not mapped
  - **Solution**: Check `config/field_config.json` for mandatory fields and ensure all are mapped

- **Error**: "Invalid field mapping"
  - **Cause**: Source field doesn't exist in input file
  - **Solution**: Verify field names match exactly with source file

#### 4. Processing Issues
- **Error**: "Empty rows detected"
  - **Cause**: Input file contains empty rows
  - **Solution**: Clean input data or use `skipEmptyRows` parameter

- **Error**: "Memory limit exceeded"
  - **Cause**: File processing requires too much memory
  - **Solution**: Process file in smaller chunks or increase server memory

#### 5. API Issues
- **Error**: "405 Method Not Allowed"
  - **Cause**: Wrong HTTP method used
  - **Solution**: Use POST for /process, GET for /config

- **Error**: "400 Bad Request"
  - **Cause**: Malformed JSON in mappings
  - **Solution**: Validate JSON format of mappings parameter

### Service Health Checks
1. Check service is running: `curl http://localhost:8080/`
2. Verify API endpoints: `curl http://localhost:8080/api/v1/config -H "X-API-Key: your-api-key"`
3. Monitor uploads directory space: `df -h ./uploads`

### Getting Help
1. Check logs for detailed error messages
2. Verify configuration in `config/field_config.json`
3. Ensure all dependencies are installed
4. Contact support with:
   - Error message
   - Input file sample
   - Field mappings used
   - Expected vs actual results

## Development
For development setup and guidelines, see [developing.md](developing.md)