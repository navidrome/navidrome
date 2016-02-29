package models

type Artist struct {
	Id     string
	Name   string
	Albums map[string]bool
}