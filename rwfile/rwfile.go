package rwfile

import (
  "bufio"
  "fmt"
  "log"
  "os"
  "strconv"
)


// ReadLines reads a whole file into memory
// and returns a slice of its lines.
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
	return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
	lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}


// WriteLines writes the lines to the given file.
func WriteLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
	return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
	fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func WriteToFile(cabRequests [4]bool, filename string) {
	
	var stringCabRequests [4]string
	
	for i := range cabRequests {
		if cabRequests[i] == false {
			stringCabRequests[i] = "0\n"
		} else {
			stringCabRequests[i] = "1\n"
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("error: ", err)
	}
	//defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range stringCabRequests {
		fmt.Fprintf(w, line)
	}
	w.Flush()

	/*if err := WriteLines(lines, "foo.out.txt"); err != nil {
	log.Fatalf("WriteLines: %s", err)
	}*/
}

func ReadFromFile(filename string) [4]int {
	var intCabRequests [4]int

	/*file, err := os.Open(filename)
	if err != nil {
		fmt.Println("error", err)
	}*/

	lines, err := ReadLines(filename)
	if err != nil {
		log.Fatalf("error: ", err)
	}
	
	for i, line := range lines {
		num, err := strconv.Atoi(line)
		intCabRequests[i] = num
		if err != nil {
			log.Fatalf("error: ", err)
		}
	}
	return intCabRequests
}