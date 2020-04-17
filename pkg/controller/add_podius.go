package controller

import (
	"github.com/wilsonianb/codius-operator/pkg/controller/podius"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, podius.Add)
}
