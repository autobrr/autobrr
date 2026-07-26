// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package newznab

import (
	"fmt"
	"regexp"
	"strconv"
)

type Category struct {
	ID            int        `xml:"id,attr"`
	Name          string     `xml:"name,attr"`
	SubCategories []Category `xml:"subcat"`
}

func (c Category) String() string {
	return fmt.Sprintf("%s[%d]", c.Name, c.ID)
}

var newzCategory = regexp.MustCompile(`(?m)(.+)\[(.+)\]`)

func (c Category) FromString(str string) {
	match := newzCategory.FindAllString(str, -1)

	c.Name = match[1]
	c.ID, _ = strconv.Atoi(match[2])
}

const (
	CustomCategoryOffset = 100000
)

// Categories from the Newznab spec
// https://github.com/nZEDb/nZEDb/blob/0.x/docs/newznab_api_specification.txt#L627
var (
	CategoryOther              = Category{0, "Other", nil}
	CategoryOther_Misc         = Category{10, "Other/Misc", nil}
	CategoryOther_Hashed       = Category{20, "Other/Hashed", nil}
	CategoryConsole            = Category{1000, "Console", nil}
	CategoryConsole_NDS        = Category{1010, "Console/NDS", nil}
	CategoryConsole_PSP        = Category{1020, "Console/PSP", nil}
	CategoryConsole_Wii        = Category{1030, "Console/Wii", nil}
	CategoryConsole_XBOX       = Category{1040, "Console/Xbox", nil}
	CategoryConsole_XBOX360    = Category{1050, "Console/Xbox360", nil}
	CategoryConsole_WiiwareVC  = Category{1060, "Console/Wiiware/V", nil}
	CategoryConsole_XBOX360DLC = Category{1070, "Console/Xbox360", nil}
	CategoryConsole_PS3        = Category{1080, "Console/PS3", nil}
	CategoryConsole_Other      = Category{1999, "Console/Other", nil}
	CategoryConsole_3DS        = Category{1110, "Console/3DS", nil}
	CategoryConsole_PSVita     = Category{1120, "Console/PS Vita", nil}
	CategoryConsole_WiiU       = Category{1130, "Console/WiiU", nil}
	CategoryConsole_XBOXOne    = Category{1140, "Console/XboxOne", nil}
	CategoryConsole_PS4        = Category{1180, "Console/PS4", nil}
	CategoryMovies             = Category{2000, "Movies", nil}
	CategoryMovies_Foreign     = Category{2010, "Movies/Foreign", nil}
	CategoryMovies_Other       = Category{2020, "Movies/Other", nil}
	CategoryMovies_SD          = Category{2030, "Movies/SD", nil}
	CategoryMovies_HD          = Category{2040, "Movies/HD", nil}
	CategoryMovies_3D          = Category{2050, "Movies/3D", nil}
	CategoryMovies_BluRay      = Category{2060, "Movies/BluRay", nil}
	CategoryMovies_DVD         = Category{2070, "Movies/DVD", nil}
	CategoryMovies_WEBDL       = Category{2080, "Movies/WEBDL", nil}
	CategoryAudio              = Category{3000, "Audio", nil}
	CategoryAudio_MP3          = Category{3010, "Audio/MP3", nil}
	CategoryAudio_Video        = Category{3020, "Audio/Video", nil}
	CategoryAudio_Audiobook    = Category{3030, "Audio/Audiobook", nil}
	CategoryAudio_Lossless     = Category{3040, "Audio/Lossless", nil}
	CategoryAudio_Other        = Category{3999, "Audio/Other", nil}
	CategoryAudio_Foreign      = Category{3060, "Audio/Foreign", nil}
	CategoryPC                 = Category{4000, "PC", nil}
	CategoryPC_0day            = Category{4010, "PC/0day", nil}
	CategoryPC_ISO             = Category{4020, "PC/ISO", nil}
	CategoryPC_Mac             = Category{4030, "PC/Mac", nil}
	CategoryPC_PhoneOther      = Category{4040, "PC/Phone-Other", nil}
	CategoryPC_Games           = Category{4050, "PC/Games", nil}
	CategoryPC_PhoneIOS        = Category{4060, "PC/Phone-IOS", nil}
	CategoryPC_PhoneAndroid    = Category{4070, "PC/Phone-Android", nil}
	CategoryTV                 = Category{5000, "TV", nil}
	CategoryTV_WEBDL           = Category{5010, "TV/WEB-DL", nil}
	CategoryTV_FOREIGN         = Category{5020, "TV/Foreign", nil}
	CategoryTV_SD              = Category{5030, "TV/SD", nil}
	CategoryTV_HD              = Category{5040, "TV/HD", nil}
	CategoryTV_Other           = Category{5999, "TV/Other", nil}
	CategoryTV_Sport           = Category{5060, "TV/Sport", nil}
	CategoryTV_Anime           = Category{5070, "TV/Anime", nil}
	CategoryTV_Documentary     = Category{5080, "TV/Documentary", nil}
	CategoryXXX                = Category{6000, "XXX", nil}
	CategoryXXX_DVD            = Category{6010, "XXX/DVD", nil}
	CategoryXXX_WMV            = Category{6020, "XXX/WMV", nil}
	CategoryXXX_XviD           = Category{6030, "XXX/XviD", nil}
	CategoryXXX_x264           = Category{6040, "XXX/x264", nil}
	CategoryXXX_Other          = Category{6999, "XXX/Other", nil}
	CategoryXXX_Imageset       = Category{6060, "XXX/Imageset", nil}
	CategoryXXX_Packs          = Category{6070, "XXX/Packs", nil}
	CategoryBooks              = Category{7000, "Books", nil}
	CategoryBooks_Magazines    = Category{7010, "Books/Magazines", nil}
	CategoryBooks_Ebook        = Category{7020, "Books/Ebook", nil}
	CategoryBooks_Comics       = Category{7030, "Books/Comics", nil}
	CategoryBooks_Technical    = Category{7040, "Books/Technical", nil}
	CategoryBooks_Foreign      = Category{7060, "Books/Foreign", nil}
	CategoryBooks_Unknown      = Category{7999, "Books/Unknown", nil}
)

var AllCategories = Categories{
	CategoryOther,
	CategoryOther_Misc,
	CategoryOther_Hashed,
	CategoryConsole,
	CategoryConsole_NDS,
	CategoryConsole_PSP,
	CategoryConsole_Wii,
	CategoryConsole_XBOX,
	CategoryConsole_XBOX360,
	CategoryConsole_WiiwareVC,
	CategoryConsole_XBOX360DLC,
	CategoryConsole_PS3,
	CategoryConsole_Other,
	CategoryConsole_3DS,
	CategoryConsole_PSVita,
	CategoryConsole_WiiU,
	CategoryConsole_XBOXOne,
	CategoryConsole_PS4,
	CategoryMovies,
	CategoryMovies_Foreign,
	CategoryMovies_Other,
	CategoryMovies_SD,
	CategoryMovies_HD,
	CategoryMovies_3D,
	CategoryMovies_BluRay,
	CategoryMovies_DVD,
	CategoryMovies_WEBDL,
	CategoryAudio,
	CategoryAudio_MP3,
	CategoryAudio_Video,
	CategoryAudio_Audiobook,
	CategoryAudio_Lossless,
	CategoryAudio_Other,
	CategoryAudio_Foreign,
	CategoryPC,
	CategoryPC_0day,
	CategoryPC_ISO,
	CategoryPC_Mac,
	CategoryPC_PhoneOther,
	CategoryPC_Games,
	CategoryPC_PhoneIOS,
	CategoryPC_PhoneAndroid,
	CategoryTV,
	CategoryTV_WEBDL,
	CategoryTV_FOREIGN,
	CategoryTV_SD,
	CategoryTV_HD,
	CategoryTV_Other,
	CategoryTV_Sport,
	CategoryTV_Anime,
	CategoryTV_Documentary,
	CategoryXXX,
	CategoryXXX_DVD,
	CategoryXXX_WMV,
	CategoryXXX_XviD,
	CategoryXXX_x264,
	CategoryXXX_Other,
	CategoryXXX_Imageset,
	CategoryXXX_Packs,
	CategoryBooks,
	CategoryBooks_Magazines,
	CategoryBooks_Ebook,
	CategoryBooks_Comics,
	CategoryBooks_Technical,
	CategoryBooks_Foreign,
	CategoryBooks_Unknown,
}

func ParentCategory(c Category) Category {
	switch {
	case c.ID < 1000:
		return CategoryOther
	case c.ID < 2000:
		return CategoryConsole
	case c.ID < 3000:
		return CategoryMovies
	case c.ID < 4000:
		return CategoryAudio
	case c.ID < 5000:
		return CategoryPC
	case c.ID < 6000:
		return CategoryTV
	case c.ID < 7000:
		return CategoryXXX
	case c.ID < 8000:
		return CategoryBooks
	}
	return CategoryOther
}

type Categories []Category

func (slice Categories) Subset(ids ...int) Categories {
	cats := Categories{}

	for _, cat := range AllCategories {
		for _, id := range ids {
			if cat.ID == id {
				cats = append(cats, cat)
			}
		}
	}

	return cats
}

func (slice Categories) Len() int {
	return len(slice)
}

func (slice Categories) Less(i, j int) bool {
	return slice[i].ID < slice[j].ID
}

func (slice Categories) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
