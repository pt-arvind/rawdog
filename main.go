package main

import "flag"

func main() {

	var isMockPtr *bool = nil
	var isServicePtr *bool = nil

	isMockPtr = flag.Bool("m", false, "makes mocks from interfaces. rawdog -m <infile> <outfile to generate>")
	isServicePtr = flag.Bool("s", false, "makes service from model file. rawdog -s <model file> <service file to generate>")

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

	// //FIXME: temporary
	// makeService("./test/account.go", "./test/account_service.go")

	flag.Usage()
}
