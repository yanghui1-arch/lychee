package models

import "embed"

// OCR contains the fixed PP-OCRv6 Small weights and the PP-DocLayout-S layout model.
//
//go:embed ocr/*
var OCR embed.FS
