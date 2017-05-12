package main

import "flag"

func main() {

	var isMockPtr *bool = nil
	var isServicePtr *bool = nil
	var isControllerPtr *bool = nil

	isMockPtr = flag.Bool("m", false, "makes mocks from interfaces. rawdog -m <infile> <outfile to generate>")
	isServicePtr = flag.Bool("s", false, "makes service from model file. rawdog -s <model file> <service file to generate>")
	isControllerPtr = flag.Bool("c", false, "creates a controller file with the standard structure. rawdog -c <name of controller> <output dir>")

	// isDBServicePtr = flag.Bool("db", false, "change me")

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

	//FIXME: temporary
	// makeController("ResourceRole", "test/")

	flag.Usage()
}
