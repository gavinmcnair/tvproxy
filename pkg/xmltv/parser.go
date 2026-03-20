package xmltv

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"time"
)

type TV struct {
	Channels   []Channel
	Programmes []Programme
}

type Channel struct {
	ID          string
	DisplayName string
	Icon        string
}

type Programme struct {
	Channel          string
	Start            time.Time
	Stop             time.Time
	Title            string
	Description      string
	Category         string
	EpisodeNum       string
	Icon             string
	Subtitle         string
	Date             string
	Language         string
	IsNew            bool
	IsPreviouslyShown bool
	Credits          string
	Rating           string
	RatingIcon       string
	StarRating       string
	SubCategories    string
	EpisodeNumSystem string
}

type xmlTV struct {
	XMLName    xml.Name       `xml:"tv"`
	Channels   []xmlChannel   `xml:"channel"`
	Programmes []xmlProgramme `xml:"programme"`
}

type xmlChannel struct {
	ID          string    `xml:"id,attr"`
	DisplayName []xmlText `xml:"display-name"`
	Icon        *xmlIcon  `xml:"icon"`
}

type xmlProgramme struct {
	Start           string          `xml:"start,attr"`
	Stop            string          `xml:"stop,attr"`
	Channel         string          `xml:"channel,attr"`
	Title           []xmlText       `xml:"title"`
	SubTitle        []xmlText       `xml:"sub-title"`
	Desc            []xmlText       `xml:"desc"`
	Date            string          `xml:"date"`
	Category        []xmlText       `xml:"category"`
	EpisodeNum      []xmlEpisode    `xml:"episode-num"`
	Icon            *xmlIcon        `xml:"icon"`
	Credits         *xmlCredits     `xml:"credits"`
	Rating          []xmlRating     `xml:"rating"`
	StarRating      []xmlStarRating `xml:"star-rating"`
	Language        []xmlText       `xml:"language"`
	PreviouslyShown *struct{}       `xml:"previously-shown"`
	New             *struct{}       `xml:"new"`
}

type xmlText struct {
	Value string `xml:",chardata"`
}

type xmlIcon struct {
	Src string `xml:"src,attr"`
}

type xmlEpisode struct {
	System string `xml:"system,attr"`
	Value  string `xml:",chardata"`
}

type xmlCredits struct {
	Directors []string `xml:"director"`
	Actors    []string `xml:"actor"`
	Writers   []string `xml:"writer"`
}

type xmlRating struct {
	System string  `xml:"system,attr"`
	Value  string  `xml:"value"`
	Icon   *xmlIcon `xml:"icon"`
}

type xmlStarRating struct {
	Value string `xml:"value"`
}

type creditsJSON struct {
	Directors []string `json:"directors,omitempty"`
	Actors    []string `json:"actors,omitempty"`
	Writers   []string `json:"writers,omitempty"`
}

const xmltvTimeFormat = "20060102150405 -0700"

func Parse(r io.Reader) (*TV, error) {
	var raw xmlTV
	if err := xml.NewDecoder(r).Decode(&raw); err != nil {
		return nil, err
	}

	tv := &TV{}
	for _, ch := range raw.Channels {
		c := Channel{ID: ch.ID}
		if len(ch.DisplayName) > 0 {
			c.DisplayName = ch.DisplayName[0].Value
		}
		if ch.Icon != nil {
			c.Icon = ch.Icon.Src
		}
		tv.Channels = append(tv.Channels, c)
	}

	for _, p := range raw.Programmes {
		prog := Programme{Channel: p.Channel}
		if len(p.Title) > 0 {
			prog.Title = p.Title[0].Value
		}
		if len(p.Desc) > 0 {
			prog.Description = p.Desc[0].Value
		}
		if len(p.Category) > 0 {
			prog.Category = p.Category[0].Value
			if len(p.Category) > 1 {
				extra := make([]string, 0, len(p.Category)-1)
				for _, c := range p.Category[1:] {
					extra = append(extra, c.Value)
				}
				if data, err := json.Marshal(extra); err == nil {
					prog.SubCategories = string(data)
				}
			}
		}
		if len(p.EpisodeNum) > 0 {
			prog.EpisodeNum = p.EpisodeNum[0].Value
			prog.EpisodeNumSystem = p.EpisodeNum[0].System
		}
		if p.Icon != nil {
			prog.Icon = p.Icon.Src
		}
		if len(p.SubTitle) > 0 {
			prog.Subtitle = p.SubTitle[0].Value
		}
		prog.Date = p.Date
		if len(p.Language) > 0 {
			prog.Language = p.Language[0].Value
		}
		prog.IsNew = p.New != nil
		prog.IsPreviouslyShown = p.PreviouslyShown != nil

		if p.Credits != nil {
			cj := creditsJSON{
				Directors: p.Credits.Directors,
				Actors:    p.Credits.Actors,
				Writers:   p.Credits.Writers,
			}
			if len(cj.Directors) > 0 || len(cj.Actors) > 0 || len(cj.Writers) > 0 {
				if data, err := json.Marshal(cj); err == nil {
					prog.Credits = string(data)
				}
			}
		}
		if len(p.Rating) > 0 {
			prog.Rating = p.Rating[0].Value
			if p.Rating[0].Icon != nil {
				prog.RatingIcon = p.Rating[0].Icon.Src
			}
		}
		if len(p.StarRating) > 0 {
			prog.StarRating = p.StarRating[0].Value
		}

		if t, err := time.Parse(xmltvTimeFormat, p.Start); err == nil {
			prog.Start = t
		}
		if t, err := time.Parse(xmltvTimeFormat, p.Stop); err == nil {
			prog.Stop = t
		}

		tv.Programmes = append(tv.Programmes, prog)
	}
	return tv, nil
}
