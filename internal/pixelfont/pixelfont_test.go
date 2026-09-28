package pixelfont

import (
	"slices"
	"testing"
)

func TestTextLaysOutGlyphsWithOneColumnGap(t *testing.T) {
	got, err := Text("KY")
	if err != nil {
		t.Fatal(err)
	}
	want := Bitmap{
		"10001010001",
		"10010010001",
		"10100001010",
		"11000000100",
		"10100000100",
		"10010000100",
		"10001000100",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Text(KY) =\n%v\nwant\n%v", got, want)
	}
}

func TestTextRejectsUnknownGlyph(t *testing.T) {
	if _, err := Text("K?"); err == nil {
		t.Fatal("expected an error for '?'")
	}
}

func TestFullNameHalfBlocks(t *testing.T) {
	name := Join(4, MustText("KUDAY"), MustText("YURTER"))
	if name.Width() != 68 {
		t.Fatalf("width = %d, want 68", name.Width())
	}
	want := []string{
		"█  ▄▀ █   █ █▀▀▀▄ ▄▀▀▀▄ █   █    █   █ █   █ █▀▀▀▄ ▀▀█▀▀ █▀▀▀▀ █▀▀▀▄",
		"█▄▀   █   █ █   █ █▄▄▄█  ▀▄▀      ▀▄▀  █   █ █▄▄▄▀   █   █▄▄▄  █▄▄▄▀",
		"█ ▀▄  █   █ █   █ █   █   █        █   █   █ █ ▀▄    █   █     █ ▀▄ ",
		"▀   ▀  ▀▀▀  ▀▀▀▀  ▀   ▀   ▀        ▀    ▀▀▀  ▀   ▀   ▀   ▀▀▀▀▀ ▀   ▀",
	}
	if got := HalfBlocks(name); !slices.Equal(got, want) {
		t.Fatalf("HalfBlocks =\n%q\nwant\n%q", got, want)
	}
}

func TestStackCentersNarrowerRows(t *testing.T) {
	got := HalfBlocks(Stack(1, MustText("KUDAY"), MustText("YURTER")))
	want := []string{
		"   █  ▄▀ █   █ █▀▀▀▄ ▄▀▀▀▄ █   █   ",
		"   █▄▀   █   █ █   █ █▄▄▄█  ▀▄▀    ",
		"   █ ▀▄  █   █ █   █ █   █   █     ",
		"   ▀   ▀  ▀▀▀  ▀▀▀▀  ▀   ▀   ▀     ",
		"█   █ █   █ █▀▀▀▄ ▀▀█▀▀ █▀▀▀▀ █▀▀▀▄",
		" ▀▄▀  █   █ █▄▄▄▀   █   █▄▄▄  █▄▄▄▀",
		"  █   █   █ █ ▀▄    █   █     █ ▀▄ ",
		"  ▀    ▀▀▀  ▀   ▀   ▀   ▀▀▀▀▀ ▀   ▀",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("stacked =\n%q\nwant\n%q", got, want)
	}
}

func TestScaleDoublesEveryPixel(t *testing.T) {
	got := HalfBlocks(Scale(MustText("KY"), 2))
	want := []string{
		"██      ██  ██      ██",
		"██    ██    ██      ██",
		"██  ██        ██  ██  ",
		"████            ██    ",
		"██  ██          ██    ",
		"██    ██        ██    ",
		"██      ██      ██    ",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("scaled =\n%q\nwant\n%q", got, want)
	}
}
