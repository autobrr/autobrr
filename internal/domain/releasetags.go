// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/moistari/rls"
	"github.com/moistari/rls/taginfo"
)

var types map[string][]*TagInfo

func init() {
	types = make(map[string][]*TagInfo)

	audio := []*TagInfo{
		{tag: "24BIT", title: "", regexp: "(?-i:24BIT)", re: nil},
		{tag: "24BIT Lossless", title: "", regexp: "(?:24BIT lossless)", re: nil},
		{tag: "16BIT", title: "", regexp: "(?-i:16BIT)", re: nil},
		{tag: "320", title: "320 Kbps", regexp: "320[\\\\-\\\\._ kbps]?", re: nil},
		{tag: "256", title: "256 Kbps", regexp: "256[\\\\-\\\\._ kbps]?", re: nil},
		{tag: "192", title: "192 Kbps", regexp: "192[\\\\-\\\\._ kbps]?", re: nil},
		{tag: "128", title: "128 Kbps", regexp: "128[\\\\-\\\\._ kbps]?", re: nil},
		{tag: "AAC-LC", title: "Advanced Audio Coding (LC)", regexp: "aac[\\-\\._ ]?lc", re: nil},
		{tag: "AAC", title: "Advanced Audio Coding (LC)", regexp: "", re: nil},
		{tag: "AC3D", title: "", regexp: "ac[\\-\\._ ]?3d", re: nil},
		{tag: "Atmos", title: "Dolby Atmos", regexp: "", re: nil},
		{tag: "APS (VBR)", title: "APS Variable Bit Rate", regexp: "", re: nil},
		{tag: "APX (VBR)", title: "APX Variable Bit Rate", regexp: "", re: nil},
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
		{tag: "Lossless", title: "", regexp: "(?i:(?:^|[^t] )Lossless)", re: nil},
		{tag: "LogScore", title: "LogScore", regexp: "log\\s?(?:\\(|\\s)(\\d+)%\\)?", re: nil},
		{tag: "Log", title: "", regexp: "(?:log)", re: nil},
		{tag: "LPCM", title: "Linear Pulse-Code Modulation", regexp: "", re: nil},
		{tag: "MP3", title: "", regexp: "", re: nil},
		{tag: "OGG", title: "", regexp: "", re: nil},
		{tag: "OPUS", title: "", regexp: "", re: nil},
		{tag: "TrueHD", title: "Dolby TrueHD", regexp: "(?:dolby[\\-\\._ ]?)?true[\\-\\._ ]?hd", re: nil},
		{tag: "VBR", title: "Variable Bit Rate", regexp: "", re: nil},
		{tag: "V0 (VBR)", title: "V0 Variable Bit Rate", regexp: "", re: nil},
		{tag: "V1 (VBR)", title: "V1 Variable Bit Rate", regexp: "", re: nil},
		{tag: "V2 (VBR)", title: "V2 Variable Bit Rate", regexp: "", re: nil},
	}
	types["audio"] = audio

	audioBitrate := []*TagInfo{
		{tag: "24BIT", title: "", regexp: "(?-i:24BIT)", re: nil},
		{tag: "24BIT Lossless", title: "", regexp: "(?:24BIT lossless)", re: nil},
		{tag: "16BIT", title: "", regexp: "(?-i:16BIT)", re: nil},
		{tag: "320", title: "320 Kbps", regexp: "320[\\\\-\\\\._ kbps]?", re: nil},
		{tag: "256", title: "256 Kbps", regexp: "256[\\\\-\\\\._ kbps]?", re: nil},
		{tag: "192", title: "192 Kbps", regexp: "192[\\\\-\\\\._ kbps]?", re: nil},
		{tag: "128", title: "128 Kbps", regexp: "128[\\\\-\\\\._ kbps]?", re: nil},
		{tag: "APS (VBR)", title: "APS Variable Bit Rate", regexp: "", re: nil},
		{tag: "APX (VBR)", title: "APX Variable Bit Rate", regexp: "", re: nil},
		{tag: "CBR", title: "Constant Bit Rate", regexp: "", re: nil},
		{tag: "Lossless", title: "", regexp: "(?i:(?:^|[^t] )Lossless)", re: nil},
		{tag: "VBR", title: "Variable Bit Rate", regexp: "", re: nil},
		{tag: "V0 (VBR)", title: "V0 Variable Bit Rate", regexp: "", re: nil},
		{tag: "V1 (VBR)", title: "V1 Variable Bit Rate", regexp: "", re: nil},
		{tag: "V2 (VBR)", title: "V2 Variable Bit Rate", regexp: "", re: nil},
	}
	types["audioBitrate"] = audioBitrate

	audioFormat := []*TagInfo{
		{tag: "AAC-LC", title: "Advanced Audio Coding (LC)", regexp: "aac[\\-\\._ ]?lc", re: nil},
		{tag: "AAC", title: "Advanced Audio Coding (LC)", regexp: "", re: nil},
		{tag: "AC3D", title: "", regexp: "ac[\\-\\._ ]?3d", re: nil},
		{tag: "Atmos", title: "Dolby Atmos", regexp: "", re: nil},
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
		{tag: "EAC3D", title: "", regexp: "", re: nil},
		{tag: "ES", title: "Dolby Digital (ES)", regexp: "(?-i:ES)", re: nil},
		{tag: "EX", title: "Dolby Digital (EX)", regexp: "(?-i:EX)", re: nil},
		{tag: "FLAC", title: "Free Lossless Audio Codec", regexp: "", re: nil},
		{tag: "LPCM", title: "Linear Pulse-Code Modulation", regexp: "", re: nil},
		{tag: "MP3", title: "", regexp: "", re: nil},
		{tag: "OGG", title: "", regexp: "", re: nil},
		{tag: "OPUS", title: "", regexp: "", re: nil},
		{tag: "TrueHD", title: "Dolby TrueHD", regexp: "(?:dolby[\\-\\._ ]?)?true[\\-\\._ ]?hd", re: nil},
	}
	types["audioFormat"] = audioFormat

	audioExtra := []*TagInfo{
		{tag: "Cue", title: "Cue File", regexp: "", re: nil},
		{tag: "Log100", title: "", regexp: "(log 100%|log \\(100%\\))", re: nil},
		{tag: "LogScore", title: "LogScore", regexp: "log\\s?(?:\\(|\\s)(\\d+)%\\)?", re: nil},
		{tag: "Log", title: "", regexp: "(?:log)", re: nil},
	}
	types["audioExtra"] = audioExtra

	bonus := []*TagInfo{
		{tag: "Freeleech", title: "Freeleech", regexp: "freeleech", re: nil},
	}
	types["bonus"] = bonus

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

	codecs := []*TagInfo{
		{tag: "DiVX.SBC", title: "DivX SBC", regexp: "(?:divx[\\-\\._ ]?)?sbc", re: nil},
		{tag: "x264.HQ", title: "x264 (HQ)", regexp: "x[\\\\-\\\\._ ]?264[\\\\-\\\\._ ]?hq", re: nil},
		{tag: "MPEG-2", title: "", regexp: "mpe?g(?:[\\-\\._ ]?2)?", re: nil},
		{tag: "H.265", title: "", regexp: "h[\\-\\._ ]?265", re: nil},
		{tag: "H.264", title: "", regexp: "h[\\-\\._ ]?264", re: nil},
		{tag: "H.263", title: "", regexp: "h[\\-\\._ ]?263", re: nil},
		{tag: "H.262", title: "", regexp: "h[\\-\\._ ]?2[26]2", re: nil},
		{tag: "H.261", title: "", regexp: "h[\\-\\._ ]?261", re: nil},
		{tag: "dxva", title: "Direct-X Video Acceleration", regexp: "", re: nil},
		{tag: "HEVC", title: "High Efficiency Video Coding", regexp: "", re: nil},
		{tag: "VC-1", title: "", regexp: "vc[\\-\\._ ]?1", re: nil},
		{tag: "x265", title: "", regexp: "x[\\-\\._ ]?265", re: nil},
		{tag: "x264", title: "", regexp: "x[\\-\\._ ]?264", re: nil},
		{tag: "XViD", title: "Xvid", regexp: "", re: nil},
		{tag: "AVC", title: "Advanced Video Coding", regexp: "avc(?:[\\-\\._ ]?1)?", re: nil},
		{tag: "VP9", title: "", regexp: "vp[\\-\\._ ]?9", re: nil},
		{tag: "VP8", title: "", regexp: "vp[\\-\\._ ]?8", re: nil},
		{tag: "VP7", title: "", regexp: "vp[\\-\\._ ]?7", re: nil},
	}
	types["codecs"] = codecs

	container := []*TagInfo{
		{tag: "avi", title: "Audio Video Interleave (avi)", regexp: "", re: nil},
		{tag: "img", title: "IMG", regexp: "", re: nil},
		{tag: "iso", title: "ISO", regexp: "\\biso\\b", re: nil},
		{tag: "mkv", title: "Matroska (mkv)", regexp: "", re: nil},
		{tag: "mov", title: "MOV", regexp: "", re: nil},
		{tag: "mp4", title: "MP4", regexp: "", re: nil},
		{tag: "mpg", title: "MPEG", regexp: "mpe?g", re: nil},
		{tag: "m2ts", title: "BluRay Disc (m2ts)", regexp: "", re: nil},
		{tag: "vob", title: "VOB", regexp: "", re: nil},
	}
	types["container"] = container

	hdr := []*TagInfo{
		{tag: "HDR10+", title: "High Dynamic Range (10-bit+)", regexp: "hdr[\\-\\.]?10\\+|10\\+[\\-\\.]?bit|hdr10plus|hi10p", re: nil},
		{tag: "HDR10", title: "High Dynamic Range (10-bit)", regexp: "hdr[\\-\\.]?10|10[\\-\\.]?bit|hi10", re: nil},
		{tag: "HDR+", title: "High Dynamic Range+", regexp: "hdr\\+", re: nil},
		{tag: "HDR", title: "High Dynamic Range", regexp: "", re: nil},
		{tag: "SDR", title: "Standard Dynamic Range", regexp: "", re: nil},
		{tag: "DV", title: "Dolby Vision", regexp: "(?i:dolby[\\-\\._ ]vision|dovi|\\Qdv\\E\\b)", re: nil},
	}
	types["hdr"] = hdr

	other := []*TagInfo{
		{tag: "HYBRID", title: "Hybrid", regexp: "", re: nil},
		{tag: "REMUX", title: "Remux", regexp: "", re: nil},
		{tag: "REPACK", title: "Repack", regexp: "repack(?:ed)?", re: nil},
		{tag: "REREPACK", title: "Rerepack", regexp: "rerepack(?:ed)?", re: nil},
	}
	types["other"] = other

	origin := []*TagInfo{
		{tag: "P2P", title: "P2P", regexp: "", re: nil},
		{tag: "Scene", title: "Scene", regexp: "", re: nil},
		{tag: "O-Scene", title: "O-Scene", regexp: "", re: nil},
		{tag: "Internal", title: "Internal", regexp: "", re: nil},
		{tag: "User", title: "User", regexp: "", re: nil},
	}
	types["origin"] = origin

	source := []*TagInfo{
		{tag: "Cassette", title: "Cassette", regexp: "", re: nil},
		{tag: "CD", title: "Compact Disc", regexp: "cd[\\-\\._ ]?(?:album)?", re: nil},
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
		{tag: "SACD", title: "Super Audio Compact Disc", regexp: "", re: nil},
		{tag: "RADIO", title: "Radio", regexp: "(?-i:R[aA]D[iI][oO])", re: nil},
		{tag: "SATRiP", title: "Satellite (rip)", regexp: "sat[\\-\\._ ]?rip", re: nil},
		{tag: "SAT", title: "Satellite Radio", regexp: "(?-i:SAT)", re: nil},
		{tag: "SBD", title: "Soundboard", regexp: "(?-i:SBD|DAB|Soundboard)", re: nil},
		{tag: "UHD.BDRiP", title: "Ultra High-Definition BluRay (rip)", regexp: "uhd[\\-\\._ ]?(?:bd)?rip", re: nil},
		{tag: "UHD.BluRay", title: "Ultra High-Definition BluRay", regexp: "uhd[\\-\\._ ]?(?:blu[\\-\\._ ]?ray|bd)", re: nil},
		{tag: "UHDTV", title: "Ultra High-Definition TV", regexp: "", re: nil},
		{tag: "UMDMOVIE", title: "Universal Media Disc Movie", regexp: "", re: nil},
		{tag: "Vinyl", title: "Vinyl", regexp: "vinyl|vl", re: nil},
		{tag: "WEB-DL", title: "Web (DL)", regexp: "web[\\-\\._ ]?dl", re: nil},
		{tag: "WEB-HD", title: "Web (HD)", regexp: "web[\\-\\._ ]?hd", re: nil},
		{tag: "WEBFLAC", title: "Web (FLAC)", regexp: "", re: nil},
		{tag: "WebHDRiP", title: "Web (HD rip)", regexp: "", re: nil},
		{tag: "WEBRiP", title: "Web (rip)", regexp: "web[\\-\\._ ]?rip", re: nil},
		{tag: "WEBSCR", title: "Web (screener)", regexp: "web[\\-\\._ ]?scr(?:eener)?", re: nil},
		{tag: "WebUHD", title: "Web (UHD)", regexp: "", re: nil},
		{tag: "WEB", title: "Web", regexp: "", re: nil},
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

	// language `(?i)\b((DK|DKSUBS|DANiSH|DUTCH|NL|NLSUBBED|ENG|FI|FLEMiSH|FiNNiSH|DE|FRENCH|GERMAN|HE|HEBREW|HebSub|HiNDi|iCELANDiC|KOR|MULTi|MULTiSUBS|NORWEGiAN|NO|NORDiC|PL|PO|POLiSH|PLDUB|RO|ROMANiAN|RUS|SPANiSH|SE|SWEDiSH|SWESUB||))\b`)
	// websites `(?i)\b((AMBC|AS|AMZN|AMC|ANPL|ATVP|iP|CORE|BCORE|CMOR|CN|CBC|CBS|CMAX|CNBC|CC|CRIT|CR|CSPN|CW|DAZN|DCU|DISC|DSCP|DSNY|DSNP|DPLY|ESPN|FOX|FUNI|PLAY|HBO|HMAX|HIST|HS|HOTSTAR|HULU|iT|MNBC|MTV|NATG|NBC|NF|NICK|NRK|PMNT|PMNP|PCOK|PBS|PBSK|PSN|QIBI|SBS|SHO|STAN|STZ|SVT|SYFY|TLC|TRVL|TUBI|TV3|TV4|TVL|VH1|VICE|VMEO|UFC|USAN|VIAP|VIAPLAY|VL|WWEN|XBOX|YHOO|YT|RED))\b`)

	for _, infos := range types {
		for _, info := range infos {
			info.re = regexp.MustCompile(`(?i)(?:` + info.RE() + `)`)
		}
	}

	var extraTagInfos = map[string][]*taginfo.Taginfo{}

	extraCollections := [][]string{
		{"4OD", "4OD", "(?-i:4OD)", "", "", ""},
		{"ABEMA", "Abema", "(?-i:ABEMA)", "", "", ""},
		{"ADN", "Animation Digital Network", "(?-i:ADN)", "", "", ""},
		{"AUBC", "Australian Broadcasting Corporation", "", "", "", ""},
		{"AUViO", "French AUViO", "(?-i:AUViO)", "", "", ""},
		{"Bilibili", "Bilibili", "(?-i:Bilibili)", "", "", ""},
		{"CRiT", "Criterion Channel", "(?-i:CRiT)", "", "", ""},
		{"FOD", "Fuji Television On Demand", "(?-i:FOD)", "", "", ""},
		{"HIDIVE", "HIDIVE", "(?-i:HIDIVE)", "", "", ""},
		{"ITVX", "ITVX aka ITV", "", "", "", ""},
		{"MA", "Movies Anywhere", "(?-i:MA)", "", "", ""},
		{"MY5", "MY5 aka Channel 5", "", "", "", ""},
		{"MyCanal", "French Groupe Canal+", "(?-i:MyCanal)", "", "", ""},
		{"NOW", "Now", "(?-i:NOW)", "", "", ""},
		{"NLZ", "Dutch NLZiet", "(?-i:NLZ|NLZiet)", "", "", ""},
		{"OViD", "OViD", "(?-i:OViD)", "", "", ""},
		{"STRP", "Star+", "(?-i:STRP)", "", "", ""},
		{"U-NEXT", "U-NEXT", "(?-i:U-NEXT)", "", "", ""},
		{"TVer", "TVer", "(?-i:TVer)", "", "", ""},
		{"TVING", "TVING", "(?-i:TVING)", "", "", ""},
		{"VIU", "VIU", "(?-i:VIU)", "", "", ""},
		{"VDL", "Videoland", "(?-i:VDL)", "", "", ""},
		{"VRV", "VRV", "(?-i:VRV)", "", "", ""},
		{"Pathe", "Pathé Thuis", "(?-i:Pathe)", "", "", ""},
		{"SALTO", "SALTO", "(?-i:SALTO)", "", "", ""},
		{"SHOWTIME", "SHOWTIME", "(?-i:SHO|SHOWTIME)", "", "", ""},
		{"SYFY", "SYFY", "(?-i:SYFY)", "", "", ""},
		{"QUIBI", "QUIBI", "(?-i:QIBI|QUIBI)", "", "", ""},
	}

	for _, collection := range extraCollections {
		inf, err := taginfo.New(collection[0], collection[1], collection[2], collection[3], collection[4], collection[5])
		if err != nil {
			//log.Fatal(err)
		}
		extraTagInfos["collection"] = append(extraTagInfos["collection"], inf)
	}

	rls.DefaultParser = rls.NewTagParser(taginfo.All(extraTagInfos), rls.DefaultLexers()...)
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

// FindMatch returns the regexp matches.
func (info *TagInfo) FindMatch(t string) []string {
	return info.re.FindStringSubmatch(t)
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
	Audio        []string
	AudioBitrate string
	AudioFormat  string
	LogScore     int
	HasLog       bool
	HasCue       bool
	Bonus        []string
	Channels     string
	Codec        string
	Container    string
	HDR          []string
	Origin       string
	Other        []string
	Resolution   string
	Source       string
}

func ParseReleaseTags(tags []string) ReleaseTags {
	releaseTags := ReleaseTags{}

	for _, tag := range tags {

		for tagType, tagInfos := range types {

			for _, info := range tagInfos {

				if info.Tag() == "LogScore" {
					m := info.FindMatch(tag)
					if len(m) == 3 {
						score, err := strconv.Atoi(m[2])
						if err != nil {
							// handle error
						}
						releaseTags.LogScore = score
					}
					continue
				}

				// check tag
				match := info.Match(tag)
				if match {
					switch tagType {
					case "audio":
						releaseTags.Audio = append(releaseTags.Audio, info.Tag())
						continue
					case "audioBitrate":
						releaseTags.AudioBitrate = info.Tag()
						continue
					case "audioFormat":
						releaseTags.AudioFormat = info.Tag()
						continue
					case "bonus":
						releaseTags.Bonus = append(releaseTags.Bonus, info.Tag())
						continue
					case "channels":
						releaseTags.Channels = info.Tag()
						break
					case "codecs":
						releaseTags.Codec = info.Tag()
						break
					case "container":
						releaseTags.Container = info.Tag()
						break
					case "hdr":
						releaseTags.HDR = append(releaseTags.HDR, info.Tag())
						continue
					case "origin":
						releaseTags.Origin = info.Tag()
						break
					case "other":
						releaseTags.Other = append(releaseTags.Other, info.Tag())
						continue
					case "source":
						releaseTags.Source = info.Tag()
						break
					case "resolution":
						releaseTags.Resolution = info.Tag()
						break
					}
					break
				}
			}
		}
	}

	return releaseTags
}

func ParseReleaseTagString(tags string) ReleaseTags {
	releaseTags := ReleaseTags{}

	if tags == "" {
		return releaseTags
	}

	for tagType, tagInfos := range types {

		for _, info := range tagInfos {
			// check tag
			match := info.Match(tags)
			if !match {
				continue
			}

			if info.Tag() == "LogScore" {
				m := info.FindMatch(tags)
				if len(m) == 2 {
					score, err := strconv.Atoi(m[1])
					if err != nil {
						// handle error
					}
					releaseTags.HasLog = true
					releaseTags.LogScore = score

					releaseTags.Audio = append(releaseTags.Audio, fmt.Sprintf("Log%d", score))
				}
				continue
			}

			switch tagType {
			case "audio":
				releaseTags.Audio = append(releaseTags.Audio, info.Tag())
				if info.Tag() == "Cue" {
					releaseTags.HasCue = true
				}
				continue
			case "audioBitrate":
				releaseTags.AudioBitrate = info.Tag()
				continue
			case "audioFormat":
				releaseTags.AudioFormat = info.Tag()
				continue
			case "bonus":
				releaseTags.Bonus = append(releaseTags.Bonus, info.Tag())
				continue
			case "channels":
				releaseTags.Channels = info.Tag()
				break
			case "codecs":
				releaseTags.Codec = info.Tag()
				break
			case "container":
				releaseTags.Container = info.Tag()
				break
			case "hdr":
				releaseTags.HDR = append(releaseTags.HDR, info.Tag())
				continue
			case "origin":
				releaseTags.Origin = info.Tag()
				break
			case "other":
				releaseTags.Other = append(releaseTags.Other, info.Tag())
				continue
			case "source":
				releaseTags.Source = info.Tag()
				break
			case "resolution":
				releaseTags.Resolution = info.Tag()
				break
			}
			break
		}

	}

	return releaseTags
}

var tagsDelimiterRegexp = regexp.MustCompile(`\s*[|/,]\s*`)

// CleanReleaseTags trim delimiters and closest space
func CleanReleaseTags(tagString string) string {
	return tagsDelimiterRegexp.ReplaceAllString(tagString, " ")
}
