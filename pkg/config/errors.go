// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package config

import "errors"

var (
	// ErrInvalidHawkeyeName is returned when the hawkeye name is invalid
	ErrInvalidHawkeyeName = errors.New("invalid hawkeye name")
	// ErrInvalidLoaderInterval is returned when the loader interval is invalid
	ErrInvalidLoaderInterval = errors.New("invalid loader interval")
	// ErrInvalidLoaderHttpURL is returned when the loader http url is invalid
	ErrInvalidLoaderHttpURL = errors.New("invalid loader http url")
	// ErrInvalidLoaderHttpRetryCount is returned when the loader http retry count is invalid
	ErrInvalidLoaderHttpRetryCount = errors.New("invalid loader http retry count")
	// ErrInvalidLoaderFilePath is returned when the loader file path is invalid
	ErrInvalidLoaderFilePath = errors.New("invalid loader file path")
)
