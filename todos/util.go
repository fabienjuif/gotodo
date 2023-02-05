package main

import "github.com/jaevor/go-nanoid"

type GenID struct {
	canonicID func() string
}

func (s *GenID) Init() error {
	canonicID, err := nanoid.Standard(21)
	if err != nil {
		return err
	}
	s.canonicID = canonicID
	return err
}

func (s *GenID) Get() string {
	return s.canonicID()
}

