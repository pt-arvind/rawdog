package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func wordsFromCtrl(controllerName string) []string {
	var words []string
	l := 0
	for s := controllerName; s != ""; s = s[l:] {
		l = strings.IndexFunc(s[1:], unicode.IsUpper) + 1
		if l <= 0 {
			l = len(s)
		}
		word := strings.ToLower(s[:l])
		words = append(words, word)
	}
	return words
}

func route(controllerName string) string {
	words := wordsFromCtrl(controllerName)
	rt := strings.Join(words, "_")

	return rt
}

func description(controllerName string) string {
	words := wordsFromCtrl(controllerName)
	desc := strings.Join(words, " ")

	return desc
}

func makeController(controllerName string, outdir string) {
	// input := modelFile
	// output := serviceFile
	rt := route(controllerName)
	desc := description(controllerName)

	ctrlTemplate := `
	package controller

	import (
		"lib/router"
		"log"
		"net/http"

		"app/webapi/logic"
		"domain"
	)

	// %ctrl_name% represents the services required for this controller.
	type %ctrl_name% struct {
		%ctrl_name% logic.I%ctrl_name%Service
		View domain.IViewAdapter
		Parser domain.IViewParser
	}

	// New%ctrl_name% returns a new instance of a %desc%
	func New%ctrl_name%(svc logic.I%ctrl_name%Service, v domain.IViewAdapter, p domain.IViewParser) *%ctrl_name% {
		s := new(%ctrl_name%)
		s.%ctrl_name% = svc
		s.View = v
		s.Parser = p
		return s
	}

	// AddRoutes adds routes for interacting with domain %ctrl_name% via the webapi.
	func (h *%ctrl_name%) AddRoutes() {
		router.Post("/%route%", h.Store)
		router.Get("/%route%", h.Index)
		router.Get("/%route%/:id", h.Show)
	}

	// Store saves a new %desc% to the database.
	func (h *%ctrl_name%) Store(w http.ResponseWriter, r *http.Request) {

	}

	// Index shows all %desc% in the system.
	func (h *%ctrl_name%) Index(w http.ResponseWriter, r *http.Request) {

	}

	// Show returns a particular %desc% with a particular ID in the system.
	func (h *%ctrl_name%) Show(w http.ResponseWriter, r *http.Request) {
		id := router.Param(r, "id")

	}
	`

	ctrl := strings.Replace(ctrlTemplate, "%ctrl_name%", controllerName, -1)
	ctrl = strings.Replace(ctrl, "%route%", rt, -1)
	ctrl = strings.Replace(ctrl, "%desc%", desc, -1)

	output := filepath.Join(outdir, rt+".go")
	file, err := os.Create(output)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString(ctrl)
}
