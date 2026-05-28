package models

import (
	"encoding/json"
	"log"

	"github.com/labstack/echo/v4"
)

func BindFromFile(c echo.Context, fileName string, v interface{}) error {
	f, err := c.FormFile(fileName)
	if err != nil {
		return err
	}
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	err = json.NewDecoder(src).Decode(v)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
