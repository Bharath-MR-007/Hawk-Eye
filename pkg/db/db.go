// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"sync"

	"github.com/Bharath-MR-007/hawk-eye/pkg/checks"
)

type DB interface {
	Save(result checks.ResultDTO)
	Get(check string) (result checks.Result, ok bool)
	List() map[string]checks.Result
}

type InMemory struct {
	// data stores a slice of results for each check name
	data sync.Map
	// maxHistory defines how many results to keep for each check
	maxHistory int
}

// NewInMemory creates a new in-memory database with history support
func NewInMemory() *InMemory {
	return &InMemory{
		data:       sync.Map{},
		maxHistory: 100, // Keep last 100 results per check
	}
}

func (i *InMemory) Save(result checks.ResultDTO) {
	val, _ := i.data.LoadOrStore(result.Name, []*checks.Result{})
	history := val.([]*checks.Result)

	// Append new result
	history = append(history, result.Result)

	// Maintain max history (simple ring buffer behavior)
	if len(history) > i.maxHistory {
		history = history[len(history)-i.maxHistory:]
	}

	i.data.Store(result.Name, history)
}

func (i *InMemory) Get(check string) (checks.Result, bool) {
	val, ok := i.data.Load(check)
	if !ok {
		return checks.Result{}, false
	}
	history := val.([]*checks.Result)
	if len(history) == 0 {
		return checks.Result{}, false
	}

	// Return the latest result
	return *history[len(history)-1], true
}

func (i *InMemory) GetHistory(check string) ([]*checks.Result, bool) {
	val, ok := i.data.Load(check)
	if !ok {
		return nil, false
	}
	return val.([]*checks.Result), true
}

// List returns the latest results for all checks
func (i *InMemory) List() map[string]checks.Result {
	results := make(map[string]checks.Result)
	i.data.Range(func(key, value any) bool {
		check := key.(string)
		history := value.([]*checks.Result)
		if len(history) > 0 {
			results[check] = *history[len(history)-1]
		}
		return true
	})

	return results
}
