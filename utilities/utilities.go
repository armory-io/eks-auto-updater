package utilities

import (
	"flag"
	"log"
	"os"
)

func Strref(s string) *string {
	return &s
}

func ValidateOrExit(parameterName string, defaultValue string, description string) *string {

	value := flag.String(parameterName, defaultValue, description)
	if (len(*value)) == 0 {
		log.Fatal("Invalid value for " + parameterName + " Must be set!")
		os.Exit(1)
	}
	return value
}
