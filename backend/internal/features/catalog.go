// Package features defines the application's release flags. Business services
// do not evaluate flags; their HTTP/module boundaries make that decision.
package features

type Key string

const (
	PlatformAI       Key = "platform_ai"
	PlatformAIPublic Key = "platform_ai_public"
	BYOKAI           Key = "byok_ai"
)

type Definition struct {
	Key         Key
	Description string
}

func Catalog() []Definition {
	return []Definition{
		{Key: PlatformAI, Description: "Platform funded AI"},
		{Key: PlatformAIPublic, Description: "Allow every signed-in user to access enabled Platform AI"},
		{Key: BYOKAI, Description: "Personal AI provider connections"},
	}
}

func known(key Key, catalog []Definition) bool {
	for _, definition := range catalog {
		if definition.Key == key {
			return true
		}
	}
	return false
}
