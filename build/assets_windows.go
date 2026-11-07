//go:build windows && amd64

package build

import _ "embed"

//go:embed windows/icon.ico
var Icon []byte
