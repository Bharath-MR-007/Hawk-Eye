package storage

import (
	"time"

	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/traceroute"
	"github.com/Bharath-MR-007/hawk-eye/pkg/db"
)

type TimeSeriesDB struct {
	db db.DB
}

func NewTimeSeriesDB(d db.DB) *TimeSeriesDB {
	return &TimeSeriesDB{db: d}
}

func (ts *TimeSeriesDB) QueryTraceroute(target string, start, end time.Time) ([]traceroute.Hop, error) {
	// Type assert to get historical data support
	inMem, ok := ts.db.(*db.InMemory)
	if !ok {
		return []traceroute.Hop{}, nil
	}

	history, ok := inMem.GetHistory("traceroute")
	if !ok {
		return []traceroute.Hop{}, nil
	}

	var allHops []traceroute.Hop
	for _, res := range history {
		// Check timeframe
		if res.Timestamp.Before(start) || res.Timestamp.After(end) {
			continue
		}

		// Data is map[string]traceroute.Result
		// Since it's saved in-memory, it will keep its original type unless marshaled.
		if data, ok := res.Data.(map[string]traceroute.Result); ok {
			if targetRes, ok := data[target]; ok {
				for _, hops := range targetRes.Hops {
					allHops = append(allHops, hops...)
				}
			}
			continue
		}

		// Fallback for interface/map types (e.g. if loaded from JSON or similar)
		if data, ok := res.Data.(map[string]interface{}); ok {
			if targetRes, ok := data[target]; ok {
				if trRes, ok := targetRes.(traceroute.Result); ok {
					for _, hops := range trRes.Hops {
						allHops = append(allHops, hops...)
					}
				} else if trMap, ok := targetRes.(map[string]interface{}); ok {
					// Deep traversal for map representation
					if hopsMap, ok := trMap["hops"].(map[int][]traceroute.Hop); ok {
						for _, hops := range hopsMap {
							allHops = append(allHops, hops...)
						}
					}
				}
			}
		}
	}

	return allHops, nil
}
