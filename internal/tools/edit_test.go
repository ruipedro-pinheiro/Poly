package tools

import "testing"

// --- normalizeWhitespace ---

func TestNormalizeWhitespace_MultipleSpaces(t *testing.T) {
	got := normalizeWhitespace("hello   world")
	want := "hello world"
	if got != want {
		t.Errorf("normalizeWhitespace(%q) = %q, want %q", "hello   world", got, want)
	}
}

func TestNormalizeWhitespace_Tabs(t *testing.T) {
	got := normalizeWhitespace("hello\tworld")
	want := "hello world"
	if got != want {
		t.Errorf("normalizeWhitespace(%q) = %q, want %q", "hello\tworld", got, want)
	}
}

func TestNormalizeWhitespace_Mixed(t *testing.T) {
	got := normalizeWhitespace("  hello   \t  world  ")
	want := "hello world"
	if got != want {
		t.Errorf("normalizeWhitespace(mixed) = %q, want %q", got, want)
	}
}

func TestNormalizeWhitespace_Multiline(t *testing.T) {
	got := normalizeWhitespace("line1\n  line2  ")
	want := "line1\nline2"
	if got != want {
		t.Errorf("normalizeWhitespace(multiline) = %q, want %q", got, want)
	}
}

func TestNormalizeWhitespace_Empty(t *testing.T) {
	got := normalizeWhitespace("")
	want := ""
	if got != want {
		t.Errorf("normalizeWhitespace('') = %q, want %q", got, want)
	}
}

// --- commonPrefixLen ---

func TestCommonPrefixLen_SameStrings(t *testing.T) {
	got := commonPrefixLen("abc", "abc")
	if got != 3 {
		t.Errorf("commonPrefixLen(abc, abc) = %d, want 3", got)
	}
}

func TestCommonPrefixLen_Partial(t *testing.T) {
	got := commonPrefixLen("abcdef", "abcxyz")
	if got != 3 {
		t.Errorf("commonPrefixLen(abcdef, abcxyz) = %d, want 3", got)
	}
}

func TestCommonPrefixLen_NoCommon(t *testing.T) {
	got := commonPrefixLen("xyz", "abc")
	if got != 0 {
		t.Errorf("commonPrefixLen(xyz, abc) = %d, want 0", got)
	}
}

func TestCommonPrefixLen_Empty(t *testing.T) {
	got := commonPrefixLen("", "abc")
	if got != 0 {
		t.Errorf("commonPrefixLen('', abc) = %d, want 0", got)
	}
}

func TestCommonPrefixLen_OneLonger(t *testing.T) {
	got := commonPrefixLen("ab", "abcdef")
	if got != 2 {
		t.Errorf("commonPrefixLen(ab, abcdef) = %d, want 2", got)
	}
}

// --- editDiffMsg ---

func TestEditDiffMsg_LinesAdded(t *testing.T) {
	got := editDiffMsg("a", "a\nb\nc")
	want := "+2 lines"
	if got != want {
		t.Errorf("editDiffMsg(add) = %q, want %q", got, want)
	}
}

func TestEditDiffMsg_LinesRemoved(t *testing.T) {
	got := editDiffMsg("a\nb\nc", "a")
	want := "-2 lines"
	if got != want {
		t.Errorf("editDiffMsg(remove) = %q, want %q", got, want)
	}
}

func TestEditDiffMsg_SameCount(t *testing.T) {
	got := editDiffMsg("a\nb", "c\nd")
	want := "same line count"
	if got != want {
		t.Errorf("editDiffMsg(same) = %q, want %q", got, want)
	}
}

// --- mapNormalizedRange ---

func TestMapNormalizedRange_SimpleMatch(t *testing.T) {
	// "hello   world" normalizes to "hello world" (11 chars)
	// Looking for normalized range [0, 11) should map to [0, 13)
	origStart, origEnd := mapNormalizedRange("hello   world", 0, 11)
	if origStart != 0 || origEnd != 13 {
		t.Errorf("mapNormalizedRange(simple) = (%d, %d), want (0, 13)", origStart, origEnd)
	}
}

func TestMapNormalizedRange_MiddleRange(t *testing.T) {
	// "aa   bb   cc" normalizes to "aa bb cc" (8 chars)
	// Normalized range for "bb" is [3,5)
	origStart, origEnd := mapNormalizedRange("aa   bb   cc", 3, 5)
	if origStart != 5 || origEnd != 7 {
		t.Errorf("mapNormalizedRange(middle) = (%d, %d), want (5, 7)", origStart, origEnd)
	}
}

func TestMapNormalizedRange_TrailingWhitespace(t *testing.T) {
	// "ab   " normalizes to "ab" (trimmed per line) actually wait...
	// normalizeWhitespace trims each line, but mapNormalizedRange works on raw bytes
	// "word   " -> normalized = "word" (4 chars if TrimSpace applied)
	// But mapNormalizedRange doesn't trim - it collapses whitespace only
	// Let's test: "a  b  " normalized has "a b " (but trimmed by normalizeWhitespace -> "a b")
	// mapNormalizedRange doesn't use normalizeWhitespace, it has its own logic
	// "a  b  " -> spaces at positions 1,2 collapse to 1 space, then 'b' at 3, then spaces 4,5
	// normalized: "a b " = 4 bytes
	// range [0,3) = "a b" -> orig should be [0, 4) (consuming trailing space of "a  b")
	origStart, origEnd := mapNormalizedRange("a  b", 0, 3)
	if origStart != 0 || origEnd != 4 {
		t.Errorf("mapNormalizedRange(trailing) = (%d, %d), want (0, 4)", origStart, origEnd)
	}
}

func TestMapNormalizedRange_NotFound(t *testing.T) {
	// Range beyond content should return (-1, -1) or partial
	origStart, origEnd := mapNormalizedRange("ab", 10, 15)
	if origStart != -1 || origEnd != -1 {
		t.Errorf("mapNormalizedRange(not found) = (%d, %d), want (-1, -1)", origStart, origEnd)
	}
}
