//go:build windows && amd64

// Package models embeds the local speech detection model and runtime.
package models

import "embed"

//go:embed silero_vad.onnx
var SileroModel []byte

//go:embed onnxruntime.dll
var ONNXRuntimeDLL []byte

//go:embed silero-LICENSE onnxruntime-LICENSE onnxruntime-NOTICES.txt
var Notices embed.FS
