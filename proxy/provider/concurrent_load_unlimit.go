//go:build 386 || amd64 || arm64 || arm64be

package provider

import "math"

const concurrentCount = math.MaxInt
