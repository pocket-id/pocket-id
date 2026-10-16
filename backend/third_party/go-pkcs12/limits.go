package pkcs12

import "fmt"

// Bound password derivation work before processing any attacker-controlled iteration count
const maxKDFIterations = 1000000

func checkKDFIterations(iterations int) error {
	if iterations < 1 || iterations > maxKDFIterations {
		return fmt.Errorf("pkcs12: KDF iteration count must be between 1 and %d", maxKDFIterations)
	}
	return nil
}
