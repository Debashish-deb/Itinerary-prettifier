package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 4 || os.Args[1] == "-h" {
		fmt.Println("itinerary usage:")
		fmt.Println("go run . ./input.txt ./output.txt ./airport-lookup.csv")
		return
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]
	airportLookupPath := os.Args[3]

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Println("Input not found")
		return
	}

	// Check if airport lookup file exists:
	if _, err := os.Stat(airportLookupPath); os.IsNotExist(err) {
		fmt.Println("Airport lookup not found")
		return
	}

	// Read airport lookup file
	airportLookup, err := readAirportLookup(airportLookupPath)
	if err != nil {
		fmt.Println("Error reading airportLookUp file: ", err)
		return
	}

	// Read input file
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Println("Error reading input file.")
		return
	}

	// Input data is processed to generate output data
	outputData, err := processInputData(string(inputData), airportLookup)
	if err != nil {
		fmt.Println("Error processing input data:", err)
		return
	}

	// Defer the writing to the end of the function
	defer func() {
		err := os.WriteFile(outputPath, []byte(outputData), 0644)
		if err != nil {
			fmt.Println("Error writing to output file.", err)
			return
		}
		fmt.Println("Output file created successfully.")
	}()
}

// Reading airport lookup file and return a map
func readAirportLookup(filepath string) (map[string]string, error) {
	airportLookup := make(map[string]string)

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	iataIndex := -1
	icaoIndex := -1
	nameIndex := -1

	// Find indices of required columns
	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "name":
			nameIndex = i
		case "icao_code":
			icaoIndex = i
		case "iata_code":
			iataIndex = i
		}
	}

	missingColumns := []string{}
	if iataIndex == -1 {
		missingColumns = append(missingColumns, "iata_code")
	}
	if icaoIndex == -1 {
		missingColumns = append(missingColumns, "icao_code")
	}
	if nameIndex == -1 {
		missingColumns = append(missingColumns, "name")
	}

	if len(missingColumns) > 0 {
		msg := fmt.Sprintf("airport lookup malformed. %s", strings.Join(missingColumns, ", "))
		return nil, fmt.Errorf(msg)
	}

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		// Check for any blank data in the required columns
		if record[iataIndex] == "" || record[icaoIndex] == "" || record[nameIndex] == "" {
			return nil, fmt.Errorf("airport lookup malformed")
		}

		// Adding data in the lookup map with IATA and ICAO codes as keys and airport names as values
		airportLookup[record[iataIndex]] = record[nameIndex]
		airportLookup[record[icaoIndex]] = record[nameIndex]
	}

	return airportLookup, nil
}

// Input data processing
func processInputData(inputData string, airportLookup map[string]string) (string, error) {
	//trimming whitespace (both leading and trailing)
	lines := strings.Split(trimWhiteSpace(inputData), "\n")
	var outputLines []string

	for _, line := range lines {
		processedLine, err := convertAirportCodes(line, airportLookup)
		if err != nil {
			return "", err
		}
		processedLine = convertISODateTime(processedLine)
		outputLines = append(outputLines, processedLine)
	}

	return strings.Join(outputLines, "\n"), nil
}

// Converting airport codes to name
func convertAirportCodes(line string, airportLookup map[string]string) (string, error) {
	var result strings.Builder
	length := len(line)
	pos := 0

	for pos < length {
		if pos+2 < length && line[pos:pos+2] == "##" {
			// Handle ICAO code
			result.WriteString(line[:pos])
			pos += 2

			if pos+4 <= length {
				code := line[pos : pos+4]
				if code == "" {
					return "", fmt.Errorf("malformed data: ICAO code is blank")
				}
				pos += 4

				if airport, found := airportLookup[code]; found {
					if airport == "" {
						return "", fmt.Errorf("malformed data: airport name for ICAO code %s is blank", code)
					}
					result.WriteString(airport)
				} else {
					result.WriteString("##" + code)
				}

				line = line[pos:]
				pos = 0
				length = len(line)
			} else {
				result.WriteString("##" + line[pos:])
				break
			}
		} else if pos+1 < length && line[pos:pos+1] == "#" {
			// Handle IATA code
			result.WriteString(line[:pos])
			pos += 1

			if pos+3 <= length {
				code := line[pos : pos+3]
				if code == "" {
					return "", fmt.Errorf("malformed data: IATA code is blank")
				}
				pos += 3

				if airport, found := airportLookup[code]; found {
					if airport == "" {
						return "", fmt.Errorf("malformed data: airport name for IATA code %s is blank", code)
					}
					result.WriteString(airport)
				} else {
					result.WriteString("#" + code)
				}

				line = line[pos:]
				pos = 0
				length = len(line)
			} else {
				result.WriteString("#" + line[pos:])
				break
			}
		} else {
			pos++
		}
	}

	result.WriteString(line)
	return result.String(), nil
}

// Converting specific format of date and time into more readable format
func convertTimeString(input string) string {
	parts := strings.Split(input, "(")
	if len(parts) != 2 {
		return input
	}

	clockStm := strings.Split(parts[0], " ")
	if len(clockStm) != 2 {
		return input
	}

	var formattedTime string
	switch clockStm[1] {
	case "T24":
		formattedTime = "15:04 (-07:00)"
	case "T12":
		formattedTime = "03:04PM (-07:00)"
	default:
		return input
	}

	timePart := parts[1][:len(parts[1])-1]
	layout := "2006-01-02T15:04-07:00"
	t, err := time.Parse(layout, timePart)
	if err != nil {
		layout = "2006-01-02T15:04Z"
		t, err = time.Parse(layout, timePart)
		if err != nil {
			return input
		}
	}

	formattedTime = t.Format(formattedTime)
	return strings.Replace(parts[0], clockStm[1], formattedTime, 1)
}

// Making ISO dates and times customer friendly
func convertISODateTime(line string) string {
	line = convertTimeString(line)

	var result strings.Builder
	pos := 0

	for {
		start := strings.Index(line[pos:], "D(")
		if start == -1 {
			result.WriteString(line[pos:])
			break
		}

		result.WriteString(line[pos : pos+start])
		pos += start

		end := strings.Index(line[pos:], ")")
		if end == -1 {
			result.WriteString(line[pos:])
			break
		}

		timeStr := line[pos+2 : pos+end]
		layouts := []string{"2006-01-02T15:04Z", "2006-01-02T15:04-07:00", "2006-01-02T15:04+02:00"}
		var t time.Time
		var err error
		for _, layout := range layouts {
			t, err = time.Parse(layout, timeStr)
			if err == nil {
				break
			}
		}
		if err != nil {
			result.WriteString(line[pos : pos+end+1])
			pos += end + 1
			continue
		}

		if pos > 0 && line[pos-1] == '.' {
			if result.Len() > 0 && result.String()[result.Len()-1] == '.' {
				result.WriteString(" ")
				pos--
			}
		}

		result.WriteString(t.Format("02 Jan 2006"))
		pos += end + 1
	}

	return result.String()
}

// vertical white space characters into new-line characters
func trimWhiteSpace(text string) string {
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\v", "\n")
	text = strings.ReplaceAll(text, "\f", "\n")

	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")

	}

	return text
}
