// This example demonstrates the usage of the fire framework to build a simple
// JSON API.
package main

import (
	"flag"

	"github.com/gonfire/fire"
	"github.com/gonfire/fire/components"
	"github.com/gonfire/fire/jsonapi"
	"github.com/gonfire/fire/model"
)

type post struct {
	model.Base `json:"-" bson:",inline" fire:"posts"`
	Title      string `json:"title" valid:"required" fire:"filterable,sortable"`
}

var inspector = flag.Bool("inspector", false, "enable inspector")

func main() {
	// parse all flags
	flag.Parse()

	// create store
	store := model.MustCreateStore("mongodb://0.0.0.0:27017/fire-test-echo")

	// create a new app
	app := fire.New()

	// create a new group
	group := jsonapi.New("api")

	// add controller
	group.Add(&jsonapi.Controller{
		Model: &post{},
		Store: store,
	})

	// mount protector
	app.Mount(components.DefaultProtector())

	// mount group
	app.Mount(group)

	// check debug mode
	if *inspector {
		// mount inspector
		app.Mount(fire.DefaultInspector())
	} else {
		// mount basic reporter
		app.Mount(fire.DefaultReporter())
	}

	// run server
	app.Start("0.0.0.0:4000")

	// yield app
	app.Yield()
}
