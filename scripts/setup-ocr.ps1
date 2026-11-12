$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$runtime = Join-Path $root 'build/bin/ocr-runtime'
python -m venv $runtime
if ($LASTEXITCODE -ne 0) { throw 'Python 3.10–3.12 is required' }
& "$runtime/Scripts/python.exe" -m pip install --timeout 180 paddleocr==3.7.0 paddlepaddle==3.3.0
if ($LASTEXITCODE -ne 0) { throw 'OCR dependency installation failed' }
Write-Host 'OCR runtime installed. PP-OCRv6 Small models are bundled inside lychee.exe; inference does not download models.'
