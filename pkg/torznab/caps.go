package torznab

import "encoding/xml"

type Server struct {
	Version   string `xml:"version,attr"`
	Title     string `xml:"title,attr"`
	Strapline string `xml:"strapline,attr"`
	Email     string `xml:"email,attr"`
	URL       string `xml:"url,attr"`
	Image     string `xml:"image,attr"`
}
type Limits struct {
	Max     string `xml:"max,attr"`
	Default string `xml:"default,attr"`
}
type Retention struct {
	Days string `xml:"days,attr"`
}

type Registration struct {
	Available string `xml:"available,attr"`
	Open      string `xml:"open,attr"`
}

type Searching struct {
	Search      Search `xml:"search"`
	TvSearch    Search `xml:"tv-search"`
	MovieSearch Search `xml:"movie-search"`
	AudioSearch Search `xml:"audio-search"`
	BookSearch  Search `xml:"book-search"`
}

type Search struct {
	Available       string `xml:"available,attr"`
	SupportedParams string `xml:"supportedParams,attr"`
}

type Categories struct {
	Category []Category `xml:"category"`
}

type Category struct {
	ID     string        `xml:"id,attr"`
	Name   string        `xml:"name,attr"`
	Subcat []SubCategory `xml:"subcat"`
}

type SubCategory struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type Groups struct {
	Group Group `xml:"group"`
}
type Group struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	Description string `xml:"description,attr"`
	Lastupdate  string `xml:"lastupdate,attr"`
}

type Genres struct {
	Genre Genre `xml:"genre"`
}

type Genre struct {
	ID         string `xml:"id,attr"`
	Categoryid string `xml:"categoryid,attr"`
	Name       string `xml:"name,attr"`
}

type Tags struct {
	Tag []Tag `xml:"tag"`
}

type Tag struct {
	Name        string `xml:"name,attr"`
	Description string `xml:"description,attr"`
}

type CapsResponse struct {
	Caps Caps `xml:"caps"`
}

type Caps struct {
	XMLName      xml.Name     `xml:"caps"`
	Server       Server       `xml:"server"`
	Limits       Limits       `xml:"limits"`
	Retention    Retention    `xml:"retention"`
	Registration Registration `xml:"registration"`
	Searching    Searching    `xml:"searching"`
	Categories   Categories   `xml:"categories"`
	Groups       Groups       `xml:"groups"`
	Genres       Genres       `xml:"genres"`
	Tags         Tags         `xml:"tags"`
}
