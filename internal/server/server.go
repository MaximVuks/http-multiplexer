package server

type Server interface {
	Run()
	Stop() error
}
