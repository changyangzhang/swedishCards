package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRenderCards_SortAndPager exercises the populated /cards table so the
// sortable headers and pagination branch (which the empty-deck path skips)
// are covered — catching missing template funcs or fields.
func TestRenderCards_SortAndPager(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	data := cardsData{
		Rows: []cardRow{
			{ID: 1, Front: "hörnet", Back: "the corner", Kind: "word", Reps: 3, DueRel: "2d"},
			{ID: 2, Front: "lediga jobb", Back: "job vacancies", Kind: "phrase", Reps: 0, DueRel: "new"},
		},
		Total:      120,
		Sort:       "front",
		Dir:        "asc",
		Page:       2,
		TotalPages: 3,
		HasPrev:    true,
		HasNext:    true,
		PrevPage:   1,
		NextPage:   3,
		RangeFrom:  51,
		RangeTo:    100,
	}
	rec := httptest.NewRecorder()
	r.Render(rec, "cards", data)

	body := rec.Body.String()
	if strings.Contains(body, "template error") {
		t.Fatalf("template error in output: %s", body)
	}
	for _, want := range []string{
		`class="sort-link"`,       // sortable headers rendered
		`?sort=front&dir=desc`,    // active asc column toggles to desc
		` ▲`,                      // active-sort arrow
		`class="pager"`,           // pager rendered
		"Page 2 / 3",              // page indicator
		"51–100 of 120",           // range summary
		`href="?page=1&sort=front&dir=asc"`, // prev preserves sort
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered /cards missing %q", want)
		}
	}
}
