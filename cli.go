package main

import "strings"

/*
This allows a user to use a comma separated list
in the golang flags pkg
*/

type StringSlice []string

func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}
