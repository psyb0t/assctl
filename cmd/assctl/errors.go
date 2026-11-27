package main

import "errors"

var errSpecTooLarge = errors.New("JSON spec exceeds 64 MiB")
