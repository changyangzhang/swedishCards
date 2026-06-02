package srs

import (
	"math"
	"time"
)

// Rating values used at the UI (4-button SM-2 variant).
const (
	RatingAgain = 1
	RatingHard  = 3
	RatingGood  = 4
	RatingEasy  = 5
)

const (
	DefaultEaseFactor = 2.5
	MinEaseFactor     = 1.3
)

type State struct {
	EaseFactor   float64
	Interval     int // days
	Repetitions  int
	Lapses       int
	DueAt        time.Time
	LastReviewed time.Time
}

// Apply runs one SM-2 step and returns the updated state. Pure function.
//
// q is the rating: RatingAgain=1, RatingHard=3, RatingGood=4, RatingEasy=5.
// Anything <3 is treated as a failed review and resets the schedule.
func Apply(s State, q int, now time.Time) State {
	out := s
	out.LastReviewed = now

	if q < 3 {
		out.Repetitions = 0
		out.Interval = 1
		out.Lapses = s.Lapses + 1
	} else {
		out.Repetitions = s.Repetitions + 1
		switch out.Repetitions {
		case 1:
			out.Interval = 1
		case 2:
			out.Interval = 6
		default:
			out.Interval = int(math.Round(float64(s.Interval) * s.EaseFactor))
			if out.Interval < 1 {
				out.Interval = 1
			}
		}
	}

	qf := float64(q)
	newEF := s.EaseFactor + (0.1 - (5-qf)*(0.08+(5-qf)*0.02))
	if newEF < MinEaseFactor {
		newEF = MinEaseFactor
	}
	out.EaseFactor = newEF
	out.DueAt = now.Add(time.Duration(out.Interval) * 24 * time.Hour)
	return out
}

// Fresh returns the starting state for a brand-new card.
func Fresh(createdAt time.Time) State {
	return State{
		EaseFactor:   DefaultEaseFactor,
		Interval:     0,
		Repetitions:  0,
		Lapses:       0,
		DueAt:        createdAt,
		LastReviewed: time.Time{},
	}
}
