package main

import (
	"fmt"
	"strconv"
)

func PadLeft(input, length int, padding string) string {
	return fmt.Sprintf("%"+padding+strconv.Itoa(length)+"d", input)
}
