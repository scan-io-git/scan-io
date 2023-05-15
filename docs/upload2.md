Upload2 is a generic command for upload scan results to defectdojo.  
Upload2 automatically create sla configuration is there are no any sla configurations.
Upload2 create new product type, specially dedicated for scanio results.
Upload2 creates product if it does not exist.


Usage example:
```
scanio upload2 -u https://defectdojo.example.com -p github.com/juice-shop/juice-shop -f ~/.scanio/results/github.com/juice-shop/juice-shop/semgrep-2023-05-13T11:09:04Z.json -t "Semgrep JSON Report"
```