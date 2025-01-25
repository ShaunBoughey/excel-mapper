// Global variables to store configuration
let fieldConfig = {
    fields: [],
    mandatoryFields: []
};

// Load configuration when the page loads
document.addEventListener('DOMContentLoaded', async () => {
    await loadConfiguration();
    setupEventListeners();
});

async function loadConfiguration() {
    try {
        const response = await fetch('/config');
        if (!response.ok) {
            throw new Error('Failed to load configuration');
        }
        fieldConfig = await response.json();
    } catch (error) {
        console.error('Error loading configuration:', error);
        alert('Failed to load field configuration. Please refresh the page.');
    }
}

function setupEventListeners() {
    document.getElementById('fileInput').addEventListener('change', handleFile, false);
    document.getElementById('mappingForm').addEventListener('submit', handleSubmit, false);
}

function handleFile(e) {
    const file = e.target.files[0];
    const submitButton = document.getElementById('submitButton');

    if (!file) {
        submitButton.disabled = true;
        return;
    }

    submitButton.disabled = false;

    const reader = new FileReader();
    reader.onload = function(event) {
        const data = new Uint8Array(event.target.result);
        const workbook = XLSX.read(data, { type: 'array' });

        const firstSheet = workbook.Sheets[workbook.SheetNames[0]];
        const headers = XLSX.utils.sheet_to_json(firstSheet, { header: 1 })[0];

        showMappingUI(headers);
    };

    reader.readAsArrayBuffer(file);
}

function showMappingUI(headers) {
    const mappingContainer = document.getElementById('mappingContainer');
    mappingContainer.innerHTML = '';

    fieldConfig.fields.forEach(field => {
        const div = document.createElement('div');
        div.classList.add('mb-3');

        const label = document.createElement('label');
        label.textContent = `Map to "${field.displayName}": `;
        label.classList.add('form-label');
        if (field.isMandatory) {
            label.innerHTML += ' <span class="text-danger">(mandatory)</span>';
        }
        
        const select = document.createElement('select');
        select.name = `mapping_${field.name}`;
        select.classList.add('form-select');
        select.dataset.mandatory = field.isMandatory ? "true" : "false";
        select.addEventListener('change', validateForm);

        const emptyOption = document.createElement('option');
        emptyOption.value = "";
        emptyOption.textContent = "-- Select Column --";
        select.appendChild(emptyOption);

        headers.forEach(header => {
            const option = document.createElement('option');
            option.value = header;
            option.textContent = header;
            select.appendChild(option);
        });

        div.appendChild(label);
        div.appendChild(select);
        mappingContainer.appendChild(div);
    });

    validateForm();
}

function validateForm() {
    const selects = document.querySelectorAll('#mappingContainer select');
    let allMandatoryMapped = true;

    selects.forEach(select => {
        if (select.dataset.mandatory === "true" && select.value === "") {
            allMandatoryMapped = false;
        }
    });

    document.getElementById('submitButton').disabled = !allMandatoryMapped;
}

function handleSubmit(e) {
    e.preventDefault();

    const formData = new FormData(document.getElementById('mappingForm'));

    fetch('/upload', {
        method: 'POST',
        body: formData
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        return response.text();
    })
    .then(data => {
        handleUploadSummary(data);
    })
    .catch(error => {
        console.error('Error:', error);
        alert('An error occurred during the upload. Please try again.');
    });
}

function handleUploadSummary(summary) {
    const resultContainer = document.getElementById('resultContainer');
    const summaryContent = document.getElementById('summaryContent');
    const downloadProcessedLink = document.getElementById('downloadProcessedLink');
    const downloadMissingLink = document.getElementById('downloadMissingLink');

    resultContainer.classList.remove('d-none');
    summaryContent.textContent = summary;

    const outputFormat = document.getElementById('outputFormat').value;
    switch(outputFormat) {
        case 'csv':
            downloadProcessedLink.href = '/download?file=processed_data.csv';
            downloadProcessedLink.download = 'processed_data.csv';
            downloadMissingLink.href = '/download?file=missing_data.csv';
            downloadMissingLink.download = 'missing_data.csv';
            downloadProcessedLink.classList.remove('d-none');
            downloadMissingLink.classList.remove('d-none');
            break;
        case 'markdown':
            downloadProcessedLink.href = '/download?file=processed_data.md';
            downloadProcessedLink.download = 'processed_data.md';
            downloadMissingLink.href = '/download?file=missing_data.md';
            downloadMissingLink.download = 'missing_data.md';
            downloadProcessedLink.classList.remove('d-none');
            downloadMissingLink.classList.remove('d-none');
            break;
        default:
            downloadProcessedLink.href = '/download?file=processed_data.xlsx';
            downloadProcessedLink.download = 'processed_data.xlsx';
            downloadProcessedLink.classList.remove('d-none');
            downloadMissingLink.classList.add('d-none');
    }
}

// Theme toggle functionality
document.addEventListener('DOMContentLoaded', () => {
    // Check for saved theme preference
    const savedTheme = localStorage.getItem('theme') || 'light';
    document.documentElement.setAttribute('data-bs-theme', savedTheme);
    updateThemeButton(savedTheme);

    // Theme toggle functionality
    const themeToggle = document.getElementById('themeToggle');
    themeToggle.addEventListener('click', () => {
        const currentTheme = document.documentElement.getAttribute('data-bs-theme');
        const newTheme = currentTheme === 'light' ? 'dark' : 'light';
        
        document.documentElement.setAttribute('data-bs-theme', newTheme);
        localStorage.setItem('theme', newTheme);
        updateThemeButton(newTheme);
    });
});

function updateThemeButton(theme) {
    const themeIcon = document.getElementById('themeIcon');
    const themeText = document.getElementById('themeText');
    
    if (theme === 'dark') {
        themeIcon.textContent = '🌜';
        themeText.textContent = 'Light Mode';
    } else {
        themeIcon.textContent = '🌞';
        themeText.textContent = 'Dark Mode';
    }
} 