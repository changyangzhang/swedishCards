package srs

import (
	"math"
	"testing"
	"time"
)

var ref = time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)

func TestApply_GoodCadence(t *testing.T) {
	s := Fresh(ref)

	s = Apply(s, RatingGood, ref)
	if s.Repetitions != 1 || s.Interval != 1 {
		t.Errorf("after 1st Good: reps=%d interval=%d, want 1/1", s.Repetitions, s.Interval)
	}
	if math.Abs(s.EaseFactor-2.5) > 0.0001 {
		t.Errorf("after 1st Good: ef=%v, want 2.5", s.EaseFactor)
	}

	s = Apply(s, RatingGood, ref.Add(24*time.Hour))
	if s.Repetitions != 2 || s.Interval != 6 {
		t.Errorf("after 2nd Good: reps=%d interval=%d, want 2/6", s.Repetitions, s.Interval)
	}

	s = Apply(s, RatingGood, ref.Add(7*24*time.Hour))
	// 6 * 2.5 = 15
	if s.Interval != 15 {
		t.Errorf("after 3rd Good: interval=%d, want 15", s.Interval)
	}
	if s.Repetitions != 3 {
		t.Errorf("after 3rd Good: reps=%d, want 3", s.Repetitions)
	}
}

func TestApply_AgainResetsAndDropsEF(t *testing.T) {
	s := Fresh(ref)
	s = Apply(s, RatingGood, ref)
	s = Apply(s, RatingGood, ref.Add(24*time.Hour))
	prevEF := s.EaseFactor
	prevLapses := s.Lapses

	s = Apply(s, RatingAgain, ref.Add(2*24*time.Hour))

	if s.Repetitions != 0 {
		t.Errorf("after Again: reps=%d, want 0", s.Repetitions)
	}
	if s.Interval != 1 {
		t.Errorf("after Again: interval=%d, want 1", s.Interval)
	}
	if s.Lapses != prevLapses+1 {
		t.Errorf("after Again: lapses=%d, want %d", s.Lapses, prevLapses+1)
	}
	if s.EaseFactor >= prevEF {
		t.Errorf("after Again: ef=%v should have dropped from %v", s.EaseFactor, prevEF)
	}
}

func TestApply_EaseFactorFloor(t *testing.T) {
	s := Fresh(ref)
	// Pummel it with Agains
	for i := 0; i < 20; i++ {
		s = Apply(s, RatingAgain, ref)
	}
	if s.EaseFactor < MinEaseFactor {
		t.Errorf("ef=%v below floor %v", s.EaseFactor, MinEaseFactor)
	}
	if math.Abs(s.EaseFactor-MinEaseFactor) > 0.0001 {
		t.Errorf("ef=%v should be exactly %v after many fails", s.EaseFactor, MinEaseFactor)
	}
}

func TestApply_HardAdvancesButLowersEF(t *testing.T) {
	s := Fresh(ref)
	s = Apply(s, RatingGood, ref)
	prevEF := s.EaseFactor

	s = Apply(s, RatingHard, ref.Add(24*time.Hour))
	if s.Repetitions != 2 {
		t.Errorf("Hard should still advance reps; got %d", s.Repetitions)
	}
	if s.EaseFactor >= prevEF {
		t.Errorf("Hard should lower ef; %v -> %v", prevEF, s.EaseFactor)
	}
}

func TestApply_EasyRaisesEF(t *testing.T) {
	s := Fresh(ref)
	s = Apply(s, RatingEasy, ref)
	if s.EaseFactor <= DefaultEaseFactor {
		t.Errorf("Easy should raise ef above %v; got %v", DefaultEaseFactor, s.EaseFactor)
	}
}

func TestApply_DueAtPushesByInterval(t *testing.T) {
	s := Fresh(ref)
	s = Apply(s, RatingGood, ref)
	want := ref.Add(24 * time.Hour)
	if !s.DueAt.Equal(want) {
		t.Errorf("due_at=%v, want %v", s.DueAt, want)
	}
}
