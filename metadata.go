package main

import "fmt"

type BadFileMetadatError struct {
	Info      *File
	JsonError error
}

func (b *BadFileMetadatError) Error() string {
	return fmt.Sprintf("%s\n\n%s\n", b.JsonError, spd.Sdump(b.Info))
}

type BadFileMetadatErrors []error
