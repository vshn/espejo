package controller

import (
	"github.com/vshn/espejo/pkg/controller/syncconfig"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, syncconfig.Add)
}
