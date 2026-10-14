// studio-api starts the Studio SDK and browser host.
package main

import (
	"github.com/viant/datly-studio/app/studioapi"
	_ "github.com/viant/datly-studio/examples/predicates"
)

func main() { studioapi.Run() }
