package datadir

import "runtime"

var pluginLockFilePathElements = [][]string{
	// Terraform >= 0.14
	{".terraform.lock.hcl"},
	// Terraform >= v0.13
	{DataDirName, "plugins", "selections.json"},
	// Terraform >= v0.12
	{DataDirName, "plugins", runtime.GOOS + "_" + runtime.GOARCH, "lock.json"},
}
