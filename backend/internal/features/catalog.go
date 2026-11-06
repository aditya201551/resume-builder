// Package features defines the application's release flags. Business services
// do not evaluate flags; their HTTP/module boundaries make that decision.
package features

type Key string

type Definition struct {
	Key         Key
	Description string
}

func Catalog() []Definition {
	return []Definition{}
}

func Known(key Key) bool {
	return known(key, Catalog())
}

func known(key Key, catalog []Definition) bool {
	for _, definition := range catalog {
		if definition.Key == key {
			return true
		}
	}
	return false
}
