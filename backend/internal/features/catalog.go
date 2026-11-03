// Package features defines the application's release flags. Business services
// do not evaluate flags; their HTTP/module boundaries make that decision.
package features

type Key string

const Jobs Key = "jobs.enabled"

type Definition struct {
	Key         Key
	Description string
}

func Catalog() []Definition {
	return []Definition{{Jobs, "Job discovery, saved jobs, and the job-search assistant"}}
}

func Known(key Key) bool {
	for _, definition := range Catalog() {
		if definition.Key == key {
			return true
		}
	}
	return false
}
