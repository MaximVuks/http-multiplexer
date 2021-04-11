package main

import "github.com/maximvuks/http-multiplexer/internal/application"

func main()  {
	app := application.New()
	app.Run()
}
