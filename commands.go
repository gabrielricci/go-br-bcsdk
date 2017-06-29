package main

type CommandResponse struct {
	CommandName  string
	Acknowledged bool
	ResponseCode string
	Parameters   []string
}
