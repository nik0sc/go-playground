Stuff in here probably won't compile, or if it does,
it will probably immediately throw a SIGSEGV or deadlock.
The special build tag `XXXnobuild` is used to exclude Go
source files in here from being compiled.

If you really want to try, pass `-tags XXXnobuild` to 
`go build`, `go run` or (why??) `go install`.