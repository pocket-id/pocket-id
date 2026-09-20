//go:build unit

// This file contains utils for unit tests and it's only built when the "unit" tag is set

package common

// SetFrancisAddresses sets a value for francisAddresses
func (c *EnvConfigSchema) SetFrancisAddresses(v []string) {
	c.francisAddresses = v
}
