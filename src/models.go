package main

import "time"

type PlayDate struct {
	id          int
	game        string
	createdDate time.Time
	date        time.Time
}

type Player struct {
	id          int
	name        string
	createdDate time.Time
}
