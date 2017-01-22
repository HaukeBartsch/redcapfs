package utils

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/tealeg/xlsx"
)

// WriteAsCsv exports the data as a csv file to the file system
func WriteAsCsv(what []map[string]string, path string) {
	// check if we got something back for writing
	if len(what) < 2 {
		vals, err := json.Marshal(what)
		if err != nil {
			fmt.Println("Error: could not convert mapping into json for print")
			return
		}
		fmt.Println("Error: we expect at least a header and some values from this call but got ", string(vals))
		return
	}

	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Cannot create file", err)
		return
	}
	defer file.Close()

	w := csv.NewWriter(file)

	header := make([]string, len(what[0]))
	i := 0
	for k := range what[0] {
		header[i] = k
		i++
	}

	if err := w.Write(header); err != nil {
		//write failed do something
		fmt.Println("Error: could not write header to file")
	}
	keys := make([]int, len(what))
	i = 0
	for k := range what {
		keys[i] = k
		i++
	}
	sort.Ints(keys)

	for _, kk := range keys {
		k := what[kk]
		values := make([]string, len(what[0]))
		i = 0
		for _ = range header {
			values[i] = k[header[i]]
			i++
		}
		if err := w.Write(values); err != nil {
			//write failed do something
		}
	}
	defer w.Flush()
}

// WriteAsJson exports the data as a json file to the file system
func WriteAsJson(what []map[string]string, path string) {
	b, err := json.MarshalIndent(what, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(path, b, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

// WriteAsExcel export the data as an excel file
func WriteAsExcel(what []map[string]string, path string) {
	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell
	var err error

	file = xlsx.NewFile()
	sheet, err = file.AddSheet("ABCD")
	if err != nil {
		fmt.Printf(err.Error())
	}
	//row = sheet.AddRow()
	//cell = row.AddCell()
	//cell.Value = "I am a cell!"

	if len(what) == 0 {
		fmt.Printf("Error: what does not contain an array %s, cannot save this value to %s", what, path)
		return
	}

	header := make([]string, len(what[0]))
	row = sheet.AddRow()
	i := 0
	for k := range what[0] {
		header[i] = k
		i++
		cell = row.AddCell()
		cell.Value = k
	}

	for _, k := range what {
		row = sheet.AddRow()
		for _, k2 := range header {
			cell = row.AddCell()
			cell.Value = k[k2]
		}
	}

	err = file.Save(path)
	if err != nil {
		fmt.Printf(err.Error())
	}
}
