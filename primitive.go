package main

//from https://tour.golang.org/basics/11
var AllPrimitives = []string{"bool", "string",
	"int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint32", "uint64", "uintptr",
	"byte",
	"rune",
	"float32", "float64",
	"complex64", "complex128",
	"error",
	"interface"}

func IsPrimitive(vType string) bool {
	for _, primitive := range AllPrimitives {
		if vType == primitive {
			return true
		}
	}
	return false
}
