package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func FailOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func GetEnvVar(envVar string) (value string) {
	err := godotenv.Load("stack.env")
	FailOnError(err, "Error loading stack.env file")

	return os.Getenv(envVar)
}
