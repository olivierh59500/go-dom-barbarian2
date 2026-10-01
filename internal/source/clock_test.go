package source

import (
	"compress/gzip"
	"encoding/csv"
	"github.com/olivierh59500/go-dom-barbarian2/assets"
	"os"
	"strconv"
	"testing"
)

func testClock(t *testing.T) *Clock {
	t.Helper()
	names := []string{"entrance-y.bin", "entrance-frames.bin", "scroll-y.bin", "profiles.bin"}
	var data [4][]byte
	for i, n := range names {
		var err error
		data[i], err = assets.Files.ReadFile("original/" + n)
		if err != nil {
			t.Fatal(err)
		}
	}
	c, err := NewClock(data[0], data[1], data[2], data[3])
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// The fixture runs the original controller with fixed YM voice levels. Drawing
// subroutines are skipped; counters, self-modifying cues and volumes execute.
func TestClockMatchesFiveThousandOriginalControllerFrames(t *testing.T) {
	f, err := os.Open("testdata/native-controller.csv.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	rows, err := csv.NewReader(z).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5001 {
		t.Fatal("incomplete native fixture")
	}
	c := testClock(t)
	c.Stage = 2
	c.Tick = EntranceTicks + CurtainTicks
	for _, row := range rows[1:] {
		c.Step([3]int{6, 9, 12})
		fields := []int{c.MainTick, c.ScrollY, c.Profile, c.ProfileTick, c.ProfileLimit, c.StartTick, c.WaitTick}
		columns := []int{0, 3, 4, 5, 6, 7, 8}
		for i, v := range fields {
			want, _ := strconv.Atoi(row[columns[i]])
			if v != want {
				t.Fatalf("tick %d field %d: got %d, want %d", c.MainTick, columns[i], v, want)
			}
		}
		moving, _ := strconv.Atoi(row[9])
		if c.Moving != (moving != 0) {
			t.Fatalf("tick %d changed background cue", c.MainTick)
		}
		for i, v := range c.Levels {
			want, _ := strconv.Atoi(row[11+i])
			if v != want {
				t.Fatalf("tick %d voice %d: got %d, want %d", c.MainTick, i, v, want)
			}
		}
	}
}
func TestIntroHasNativeTablesAndNoPerStepAllocation(t *testing.T) {
	c := testClock(t)
	for tick := 1; tick <= EntranceTicks+CurtainTicks; tick++ {
		c.Step([3]int{})
	}
	if c.Stage != 1 || c.EntryY != 84 || c.EntryFrame != 8 {
		t.Fatalf("incorrect entrance or curtain phase: %+v", c)
	}
	c.Step([3]int{})
	if c.Stage != 2 || c.MainTick != 1 {
		t.Fatal("main scene handoff changed")
	}
	if n := testing.AllocsPerRun(1000, func() { c.Step([3]int{6, 9, 12}) }); n != 0 {
		t.Fatalf("controller allocated %g objects", n)
	}
}
