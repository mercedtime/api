package main

import (
	"encoding/csv"
	"flag"
	"os"
	"path/filepath"
	"reflect"

	"github.com/mercedtime/api/db/models"
)

var csvOutDir = "data"

func init() {
	flag.StringVar(&csvOutDir, "out", csvOutDir, "output directory for csv files")
}

func csvfile(name string) (*os.File, error) {
	return os.OpenFile(filepath.Join(csvOutDir, name), os.O_CREATE|os.O_WRONLY, 0644)
}

func writeCSVFile(name string, data []interface{}) error {
	f, err := csvfile(name)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for _, c := range data {
		row, err := models.ToCSVRow(c)
		if err != nil {
			return err
		}
		if err = w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

// InterfaceSlice converts any slice into a slice of interfaces
func interfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}
	return ret
}
