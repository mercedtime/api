package main

import (
	"encoding/csv"
	"flag"
	"os"
	"path/filepath"

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
