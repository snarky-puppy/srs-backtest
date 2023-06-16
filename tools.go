//go:build tools

// See:
// https://marcofranssen.nl/manage-go-tools-via-go-modules

package internal

import (
	_ "github.com/rinchsan/gosimports/cmd/gosimports"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
