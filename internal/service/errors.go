package service

import "errors"

var (
	ErrInvalidLinkFormat = errors.New("invalid link format, expected format is <id>@<version>")
)
