package main

import (
	"flag"
	"io/ioutil"
	"strings"
)

func main() {

	var isMockPtr *bool = nil
	var isServicePtr *bool = nil
	var isControllerPtr *bool = nil
	var isDBServicePtr *bool = nil
	var isDBServiceDir *bool = nil
	var isDBTestPtr *bool = nil
	var isDBTestDir *bool = nil

	isMockPtr = flag.Bool("m", false, "makes mocks from interfaces. rawdog -m <infile> <outfile to generate>")
	isServicePtr = flag.Bool("s", false, "makes service from model file. rawdog -s <model file> <service file to generate>")
	isControllerPtr = flag.Bool("c", false, "creates a controller file with the standard structure. rawdog -c <name of controller> <output dir>")

	isDBServicePtr = flag.Bool("db", false, "makes queries from top of db model file (structs). rawdog -db <model file> ")
	isDBServiceDir = flag.Bool("dbDir", false, "makes queries for all db models in the dir (structs). rawdog -db <dir>")
	isDBTestPtr = flag.Bool("dbt", false, "makes tests from top of db model file (structs). rawdog -db <model file> ")
	isDBTestDir = flag.Bool("dbtDir", false, "makes test for all db models in the dir (structs). rawdog -db <dir>")

	flag.Parse()

	files := flag.Args()

	if *isMockPtr {
		if len(files) != 2 {
			flag.Usage()
		} else {
			input := files[0]
			output := files[1]
			makeMocks(input, output)
		}
		return
	}
	if *isServicePtr {
		if len(files) != 2 {
			flag.Usage()
		} else {
			input := files[0]
			output := files[1]
			makeService(input, output)
		}
		return
	}

	if *isControllerPtr {
		if len(files) != 2 {
			flag.Usage()
		} else {
			// not actually files
			input := files[0]
			outputDir := files[1]
			makeController(input, outputDir)
		}
		return
	}

	if *isDBServicePtr {
		input := files[0]
		output := input[0:len(input)-3] + "_generatedQueries.go"
		//log.Println(output)
		makeDBService(input, output)

		return
	}

	if *isDBTestPtr {
		input := files[0]
		output := input[0:len(input)-3] + "_generated_test.go"
		//log.Println(output)
		makeDBTests(input, output)

		return
	}

	if *isDBServiceDir {
		dirFiles, err := ioutil.ReadDir(files[0])

		if err != nil {
			//log.Println(err)
			flag.Usage()
			return
		}

		var input string
		for _, file := range dirFiles {
			if file.Name() != ".DS_Store" && file.Name() != "transactor.go" && !strings.HasSuffix(file.Name(), "_generatedQueries.go") && !strings.HasSuffix(file.Name(), "_test.go") {
				//log.Println(file.Name())
				input = files[0] + "/" + file.Name()
				output := input[0:len(input)-3] + "_generatedQueries.go"
				makeDBService(input, output)
			}
		}
		return
	}

	if *isDBTestDir {
		dirFiles, err := ioutil.ReadDir(files[0])

		if err != nil {
			//log.Println(err)
			flag.Usage()
			return
		}

		var input string
		for _, file := range dirFiles {
			if file.Name() != ".DS_Store" && file.Name() != "transactor.go" && file.Name() != "interface.go" && !strings.HasSuffix(file.Name(), "_generatedTests.go") && !strings.HasSuffix(file.Name(), "_test.go") {
				//log.Println(file.Name())
				input = files[0] + "/" + file.Name()
				output := input[0:len(input)-3] + "_generated_test.go"
				makeDBTests(input, output)
			}
		}
		return
	}
	//FIXME: temporary
	// makeController("ResourceRole", "test/")

	flag.Usage()
}
