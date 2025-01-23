![Coverage](https://img.shields.io/badge/Coverage-76.4%25-brightgreen)
[![CodeFactor](https://www.codefactor.io/repository/github/shaunboughey/excel-mapper/badge)](https://www.codefactor.io/repository/github/shaunboughey/excel-mapper)

TBD info

curl -i -X POST http://localhost:8080/api/v1/process \
  -F "file=@synthetic_test_data.xlsx" \
  -F 'mappings={
    "Client_Code":"Client Code",
    "Customer_ID":"Customer ID",
    "Account_ID":"Account Number",
    "CID":"CID",
    "AID":"AID",
    "Customer_Name":"Customer Name",
    "Account_Name":"Account Name"
  }' \
  -F "outputFormat=xlsx" \
  --output processed_data.xlsx