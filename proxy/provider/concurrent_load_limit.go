//go:build !386 && !amd64 && !arm64 && !arm64be && !mipsle && !mips

package provider

const concurrentCount = 5
