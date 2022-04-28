package domain

import (
	"fmt"
	"regexp"
)

var types map[string][]*TagInfo

func init() {
	types = make(map[string][]*TagInfo)
	types["audio"] = []*TagInfo{}

	audio := []*TagInfo{
		{tag: "24BIT", title: "", regexp: "(?-i:24BIT)", re: nil},
		{tag: "24BIT Lossless", title: "", regexp: "(?-i:24BIT lossless)", re: nil},
		{tag: "16BIT", title: "", regexp: "(?-i:16BIT)", re: nil},
		{tag: "320Kbps", title: "320 Kbps", regexp: "320[\\-\\._ ]?kbps", re: nil},
		{tag: "256Kbps", title: "256 Kbps", regexp: "256[\\-\\._ ]?kbps", re: nil},
		{tag: "192Kbps", title: "192 Kbps", regexp: "192[\\-\\._ ]?kbps", re: nil},
		{tag: "128Kbps", title: "128 Kbps", regexp: "128[\\-\\._ ]?kbps", re: nil},
		{tag: "AAC-LC", title: "Advanced Audio Coding (LC)", regexp: "aac[\\-\\._ ]?lc", re: nil},
		{tag: "AAC", title: "Advanced Audio Coding (LC)", regexp: "", re: nil},
		{tag: "AC3D", title: "", regexp: "ac[\\-\\._ ]?3d", re: nil},
		{tag: "Atmos", title: "Dolby Atmos", regexp: "", re: nil},
		{tag: "CBR", title: "Constant Bit Rate", regexp: "", re: nil},
		{tag: "Cue", title: "Cue File", regexp: "", re: nil},
		{tag: "DDPA", title: "Dolby Digital+ Atmos (E-AC-3+Atmos)", regexp: "dd[p\\+]a", re: nil},
		{tag: "DDP", title: "Dolby Digital+ (E-AC-3)", regexp: "dd[p\\+]|e[\\-\\._ ]?ac3", re: nil},
		{tag: "DD", title: "Dolby Digital (AC-3)", regexp: "dd|ac3|dolby[\\-\\._ ]?digital", re: nil},
		{tag: "DTS-HD.HRA", title: "DTS (HD HRA)", regexp: "dts[\\-\\._ ]?hd[\\-\\._ ]?hra", re: nil},
		{tag: "DTS-HD.HR", title: "DTS (HD HR)", regexp: "dts[\\-\\._ ]?hd[\\-\\._ ]?hr", re: nil},
		{tag: "DTS-HD.MA", title: "DTS (HD MA)", regexp: "dts[\\-\\._ ]?hd[\\-\\._ ]?ma", re: nil},
		{tag: "DTS-HD", title: "DTS (HD)", regexp: "dts[\\-\\._ ]?hd[\\-\\._ ]?", re: nil},
		{tag: "DTS-MA", title: "DTS (MA)", regexp: "dts[\\-\\._ ]?ma[\\-\\._ ]?", re: nil},
		{tag: "DTS-X", title: "DTS (X)", regexp: "dts[\\-\\._ ]?x", re: nil},
		{tag: "DTS", title: "", regexp: "", re: nil},
		{tag: "DUAL.AUDIO", title: "Dual Audio", regexp: "dual(?:[\\-\\._ ]?audio)?", re: nil},
		{tag: "EAC3D", title: "", regexp: "", re: nil},
		{tag: "ES", title: "Dolby Digital (ES)", regexp: "(?-i:ES)", re: nil},
		{tag: "EX", title: "Dolby Digital (EX)", regexp: "(?-i:EX)", re: nil},
		{tag: "FLAC", title: "Free Lossless Audio Codec", regexp: "", re: nil},
		{tag: "LiNE", title: "Line", regexp: "(?-i:L[iI]NE)", re: nil},
		{tag: "LOSSLESS", title: "", regexp: "(?-i:LOSSLESS)", re: nil},
		{tag: "Log", title: "", regexp: "log [\\d\\.]+%", re: nil},
		{tag: "LPCM", title: "Linear Pulse-Code Modulation", regexp: "", re: nil},
		{tag: "MP3", title: "", regexp: "", re: nil},
		{tag: "OGG", title: "", regexp: "", re: nil},
		{tag: "OPUS", title: "", regexp: "", re: nil},
		{tag: "TrueHD", title: "Dolby TrueHD", regexp: "(?:dolby[\\-\\._ ]?)?true[\\-\\._ ]?hd", re: nil},
		{tag: "VBR", title: "Variable Bit Rate", regexp: "", re: nil},
	}
	types["audio"] = audio

	channels := []*TagInfo{
		{tag: "7.1", title: "", regexp: "7\\.1(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "6.1", title: "", regexp: "6\\.1(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "6.0", title: "", regexp: "6\\.0(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "5.1", title: "", regexp: "5\\.1(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "5.0", title: "", regexp: "5\\.0(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "4.1", title: "", regexp: "4\\.1(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "4.0", title: "", regexp: "4\\.0(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "3.1", title: "", regexp: "3\\.1(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "3.0", title: "", regexp: "3\\.0(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "2.1", title: "", regexp: "2\\.1(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "2.0", title: "", regexp: "2\\.0(?:[\\-\\._ ]?audios)?", re: nil},
		{tag: "1.0", title: "", regexp: "1\\.0(?:[\\-\\._ ]?audios)?", re: nil},
	}
	types["channels"] = channels

	source := []*TagInfo{
		{tag: "CD", title: "Compact Disc", regexp: "cd[\\-\\._ ]?(?:album)?", re: nil},
		{tag: "WEB", title: "Web", regexp: "", re: nil},
		{tag: "BDRiP", title: "BluRay (rip)", regexp: "b[dr]?[\\-\\._ ]?rip", re: nil},
		{tag: "BDSCR", title: "BluRay (screener)", regexp: "b[dr][\\-\\._ ]?scr(?:eener)?", re: nil},
		{tag: "BluRay3D", title: "", regexp: "blu[\\-\\._ ]?ray[\\-\\._ ]?3d|bd3d", re: nil},
		{tag: "BluRayRiP", title: "BluRay (rip)", regexp: "", re: nil},
		{tag: "BluRay", title: "", regexp: "blu[\\-\\._ ]?ray|bd", re: nil},
		{tag: "BRDRip", title: "BluRay Disc (rip)", regexp: "", re: nil},
		{tag: "DAT", title: "Datacable", regexp: "(?-i:DAT)", re: nil},
		{tag: "DVBRiP", title: "Digital Video Broadcasting (rip)", regexp: "dvb[\\-\\._ ]?rip", re: nil},
		{tag: "DVDA", title: "Audio DVD", regexp: "", re: nil},
		{tag: "DVDRiP", title: "Digital Video Disc (rip)", regexp: "dvd[\\-\\._ ]?rip", re: nil},
		{tag: "DVDSCRRiP", title: "Digital Video Disc (screener rip)", regexp: "(?:dvd[\\-\\._ ]?)?scr(?:eener)?[\\-\\._ ]?rip", re: nil},
		{tag: "DVDSCR", title: "Digital Video Disc (screener)", regexp: "(?:dvd[\\-\\._ ]?)?scr(?:eener)?", re: nil},
		{tag: "DVDS", title: "Digital Video Disc (single)", regexp: "dvds(?:ingle)?", re: nil},
		{tag: "DVD", title: "Digital Video Disc", regexp: "dvd", re: nil},
	}
	types["source"] = source

	resolution := []*TagInfo{
		{tag: "PN", title: "Selector", regexp: "p(?:al)?[\\-\\._ ]?n(?:tsc)?[\\-\\._ ]selector", re: nil},
		{tag: "DCI4K", title: "DCI 4k", regexp: "dci[\\-\\._ ]?4k|4096x2160", re: nil},
		{tag: "DCI2K", title: "DCI 2k", regexp: "dci[\\-\\._ ]?2k|2048x1080", re: nil},
		{tag: "4320p", title: "UltraHD 8K (4320p)", regexp: "4320p|7680x4320", re: nil},
		{tag: "2880p", title: "5k (2880p)", regexp: "2880p|5k|5120x2880", re: nil},
		{tag: "2160p", title: "UltraHD 4K (2160p)", regexp: "2160p|3840x2160|uhd|4k", re: nil},
		{tag: "1800p", title: "QHD+ (1800p)", regexp: "1800p|3200x1800", re: nil},
		{tag: "1440p", title: "QHD (1440p)", regexp: "1440p|2560x1440", re: nil},
		{tag: "1080p", title: "FullHD (1080p)", regexp: "1080[ip]|1920x1080", re: nil},
		{tag: "900p", title: "900[ip]|1600x900", regexp: "900[ip]|1600x900", re: nil},
		{tag: "720p", title: "HD (720p)", regexp: "720[ip]|1280x720", re: nil},
		{tag: "576p", title: "PAL (576p)", regexp: "576[ip]|720x576|pal", re: nil},
		{tag: "540p", title: "qHD (540p)", regexp: "540[ip]|960x540", re: nil},
		{tag: "480p", title: "NTSC (480p)", regexp: "480[ip]|720x480|848x480|854x480|ntsc", re: nil},
		{tag: "360p", title: "nHD (360p)", regexp: "360[ip]|640x360", re: nil},
		{tag: "$1p", title: "Other ($1p)", regexp: "([123]\\d{3})p", re: nil},
	}
	types["resolution"] = resolution

	//codecs := []*TagInfo{
	//	{tag: "", title: "", regexp: "", re: nil},
	//}
	//types["codecs"] = codecs

	for s, infos := range types {
		for _, info := range infos {
			var err error
			//if info.re, err = regexp.Compile(`(?i)^(?:` + info.RE() + `)$`); err != nil {
			if info.re, err = regexp.Compile(`(?i)(?:` + info.RE() + `)`); err != nil {
				fmt.Errorf("tag %q has invalid regexp %q\n", s, info.re)
			}
		}
	}
}

type TagInfo struct {
	tag    string
	title  string
	regexp string
	re     *regexp.Regexp
}

// Tag returns the tag info tag.
func (info *TagInfo) Tag() string {
	return info.tag
}

// Title returns the tag info title.
func (info *TagInfo) Title() string {
	return info.title
}

// Regexp returns the tag info regexp.
func (info *TagInfo) Regexp() string {
	return info.regexp
}

//// Other returns the tag info other.
//func (info *TagInfo) Other() string {
//	return info.other
//}
//
//// Type returns the tag info type.
//func (info *TagInfo) Type() int {
//	return info.typ
//}

//// Excl returns the tag info excl.
//func (info *TagInfo) Excl() bool {
//	return info.excl
//}

// RE returns the tag info regexp string.
func (info *TagInfo) RE() string {
	if info.regexp != "" {
		return info.regexp
	}
	return `\Q` + info.tag + `\E`
}

// Match matches the tag info to s.
func (info *TagInfo) Match(s string) bool {
	return info.re.MatchString(s)
}

// FindFunc is the find signature..
type FindFunc func(string) *TagInfo

// Find returns a func to find tag info.
func Find(infos ...*TagInfo) FindFunc {
	n := len(infos)
	return func(s string) *TagInfo {
		for i := 0; i < n; i++ {
			if infos[i].Match(s) {
				return infos[i]
			}
		}
		return nil
	}
}

type ReleaseTags struct {
	Audio      []string
	Channels   string
	Source     string
	Resolution string
}

func ParseReleaseTags(tags []string) ReleaseTags {
	releasetags := ReleaseTags{}

	for _, tag := range tags {
		//fmt.Printf("tag: %v\n", tag)

		for tagType, tagInfos := range types {
			//fmt.Printf("tagType: %v\n", tagType)

			for _, info := range tagInfos {
				// check tag
				match := info.Match(tag)
				if match {
					fmt.Printf("match: %v, info: %v\n", tag, info.Tag())
					switch tagType {
					case "audio":
						releasetags.Audio = append(releasetags.Audio, info.Tag())
						continue
					case "channels":
						releasetags.Channels = info.Tag()
						break
					case "source":
						releasetags.Source = info.Tag()
						break
					case "resolution":
						releasetags.Resolution = info.Tag()
						break
					}
					break
				}
			}
		}
	}

	return releasetags
}
func ParseReleaseTagString(tags string) ReleaseTags {
	releasetags := ReleaseTags{}

	for tagType, tagInfos := range types {
		//fmt.Printf("tagType: %v\n", tagType)

		for _, info := range tagInfos {
			// check tag
			match := info.Match(tags)
			if !match {
				continue
			}

			fmt.Printf("match: info: %v\n", info.Tag())
			switch tagType {
			case "audio":
				releasetags.Audio = append(releasetags.Audio, info.Tag())
			case "channels":
				releasetags.Channels = info.Tag()
				break
			case "source":
				releasetags.Source = info.Tag()
				break
			case "resolution":
				releasetags.Resolution = info.Tag()
				break
			}
			continue
		}

	}

	return releasetags
}
