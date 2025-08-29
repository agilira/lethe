module constructor_examples

go 1.23.11

require github.com/agilira/lethe v0.0.0

require github.com/agilira/go-timecache v1.0.0 // indirect

// Use local lethe module during development
replace github.com/agilira/lethe => ../..
