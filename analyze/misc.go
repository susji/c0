package analyze

func IsValidPrimitive(name string) bool {
	validprimitives := map[string]bool{
		"int":    true,
		"bool":   true,
		"char":   true,
		"void":   true,
		"string": true,
		"struct": true,
	}
	_, ok := validprimitives[name]
	return ok
}

var reserveds = map[string]bool{
	"if":          true,
	"while":       true,
	"for":         true,
	"return":      true,
	"assert":      true,
	"error":       true,
	"typedef":     true,
	"struct":      true,
	"int":         true,
	"bool":        true,
	"void":        true,
	"string":      true,
	"char":        true,
	"NULL":        true,
	"true":        true,
	"false":       true,
	"alloc":       true,
	"alloc_array": true,
	"break":       true,
	"continue":    true,
}

func IsReserved(id string) bool {
	_, ok := reserveds[id]
	return ok
}
