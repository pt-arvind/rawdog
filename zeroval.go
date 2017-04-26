package main

import (
	"fmt"
	"strings"
)

func ZeroValueFor(kind VariableKind, vType string) string {
	// https://tour.golang.org/basics/11
	if kind == Value {
		if vType == "string" {
			return "\"\""
		}
		if strings.HasPrefix(vType, "int") {
			return fmt.Sprintf("%s(0)", vType)
		}
		if strings.HasPrefix(vType, "uint") {
			return fmt.Sprintf("%s(0)", vType)
		}
		if vType == "byte" {
			return fmt.Sprintf("%s(0)", vType)
		}
		if vType == "rune" {
			return fmt.Sprintf("%s(0)", vType)
		}
		if strings.HasPrefix(vType, "float") {
			return fmt.Sprintf("%s(0)", vType)
		}
		if strings.HasPrefix(vType, "complex") {
			return fmt.Sprintf("%s(0)", vType)
		}
		if vType == "bool" {
			return "false"
		}
		if vType == "error" {
			return "nil"
		}
		return vType + "{}"
	}

	if kind == Pointer {
		return "nil"
	}

	if kind == Slice {
		return "[]"
	}

	if kind == Variadic {
		return "nil" //cant use variadic in return
	}

	if kind == GoInterface {
		return "nil"
	}

	return "nil"
}
