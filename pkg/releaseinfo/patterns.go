package releaseinfo

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
)

var patterns = []struct {
	name string
	// Use the last matching pattern. E.g. Year.
	last bool
	kind reflect.Kind
	// REs need to have 2 sub expressions (groups), the first one is "raw", and
	// the second one for the "clean" value.
	// E.g. Epiode matching on "S01E18" will result in: raw = "E18", clean = "18".
	re *regexp.Regexp
}{
	//{"season", false, reflect.Int, regexp.MustCompile(`(?i)(s?([0-9]{1,2}))[ex]`)},
	{"season", false, reflect.Int, regexp.MustCompile(`(?i)((?:S|Season\s*)(\d{1,3}))`)},
	{"episode", false, reflect.Int, regexp.MustCompile(`(?i)([ex]([0-9]{2})(?:[^0-9]|$))`)},
	{"episode", false, reflect.Int, regexp.MustCompile(`(-\s+([0-9]+)(?:[^0-9]|$))`)},
	{"year", true, reflect.Int, regexp.MustCompile(`\b(((?:19[0-9]|20[0-9])[0-9]))\b`)},

	{"resolution", false, reflect.String, regexp.MustCompile(`\b(([0-9]{3,4}p|i))\b`)},
	{"source", false, reflect.String, regexp.MustCompile(`(?i)\b(((?:PPV\.)?[HP]DTV|(?:HD)?CAM|B[DR]Rip|(?:HD-?)?TS|(?:PPV )?WEB-?DL(?: DVDRip)?|HDRip|DVDRip|DVDRIP|CamRip|WEB|W[EB]BRip|BluRay|DvDScr|telesync))\b`)},
	{"codec", false, reflect.String, regexp.MustCompile(`(?i)\b((xvid|HEVC|[hx]\.?26[45]))\b`)},
	{"container", false, reflect.String, regexp.MustCompile(`(?i)\b((MKV|AVI|MP4))\b`)},

	{"audio", false, reflect.String, regexp.MustCompile(`(?i)\b((MP3|DD5\.?1|Dual[\- ]Audio|LiNE|DTS|AAC[.-]LC|AAC(?:\.?2\.0)?|AC3(?:\.5\.1)?))\b`)},
	{"region", false, reflect.String, regexp.MustCompile(`(?i)\b(R([0-9]))\b`)},
	{"size", false, reflect.String, regexp.MustCompile(`(?i)\b((\d+(?:\.\d+)?(?:GB|MB)))\b`)},
	{"website", false, reflect.String, regexp.MustCompile(`^(\[ ?([^\]]+?) ?\])`)},
	{"language", false, reflect.String, regexp.MustCompile(`(?i)\b((rus\.eng|ita\.eng))\b`)},
	{"sbs", false, reflect.String, regexp.MustCompile(`(?i)\b(((?:Half-)?SBS))\b`)},

	{"group", false, reflect.String, regexp.MustCompile(`\b(- ?([^-]+(?:-={[^-]+-?$)?))$`)},

	{"extended", false, reflect.Bool, regexp.MustCompile(`(?i)\b(EXTENDED(:?.CUT)?)\b`)},
	{"hardcoded", false, reflect.Bool, regexp.MustCompile(`(?i)\b((HC))\b`)},

	{"proper", false, reflect.Bool, regexp.MustCompile(`(?i)\b((PROPER))\b`)},
	{"repack", false, reflect.Bool, regexp.MustCompile(`(?i)\b((REPACK))\b`)},

	{"widescreen", false, reflect.Bool, regexp.MustCompile(`(?i)\b((WS))\b`)},
	{"unrated", false, reflect.Bool, regexp.MustCompile(`(?i)\b((UNRATED))\b`)},
	{"threeD", false, reflect.Bool, regexp.MustCompile(`(?i)\b((3D))\b`)},
}

func init() {
	for _, pat := range patterns {
		if pat.re.NumSubexp() != 2 {
			fmt.Printf("Pattern %q does not have enough capture groups. want 2, got %d\n", pat.name, pat.re.NumSubexp())
			os.Exit(1)
		}
	}
}
