package core

import "context"

type placeholderError struct {
	Inner string
}

func (pe placeholderError) Error() string {
	return pe.Inner
}

type CoreError struct {
	InnerError error
	Context    string
}

func (e CoreError) Error() string {
	return e.InnerError.Error()
}

type CollectionNotFoundError struct {
	CoreError
	ManifestUid uint64
}

func (e CollectionNotFoundError) Error() string {
	return e.InnerError.Error()
}

type contextualDeadline struct {
	Message string
}

func (e contextualDeadline) Error() string {
	return e.Message
}

func (e contextualDeadline) Unwrap() error {
	return context.DeadlineExceeded
}
