package main

import (
	"fmt"
	"os"
)

func main() {
	if err := StreamSQLiteToS3(); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur backup: %v\n", err)
		os.Exit(1)
	}

	if err := RestoreFromS3(); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur restore: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Opérations terminées avec succès")
}
