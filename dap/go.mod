module github.com/arturoeanton/goja-debug-poc/dap

go 1.24.5

require github.com/dop251/goja v0.0.0-20250630131328-58d95d85e994

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	golang.org/x/text v0.3.8 // indirect
)

//replace github.com/dop251/goja => github.com/arturoeanton/goja v0.0.0-20250729040025-e2ff0c5841bb
replace github.com/dop251/goja => ../../goja/
