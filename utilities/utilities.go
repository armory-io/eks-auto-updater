package utilities

import (
	"flag"
	"log"
)

//
//func StringReference(s string) *string {
//	return &s
//}

func ValidateOrExit(parameterName string, description string) *string {

	value := flag.String(parameterName, "", description)
	if (len(*value)) == 0 {
		log.Fatal("ERROR: Invalid value for " + parameterName + " Must be set!")
	}
	return value
}
