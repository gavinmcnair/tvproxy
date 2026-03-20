package xmltv

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <channel id="channel1">
    <display-name>Channel One</display-name>
    <icon src="http://example.com/icon1.png"/>
  </channel>
  <channel id="channel2">
    <display-name>Channel Two</display-name>
  </channel>
  <programme start="20240101120000 +0000" stop="20240101130000 +0000" channel="channel1">
    <title>Test Show</title>
    <desc>A test show description</desc>
    <category>News</category>
    <episode-num system="onscreen">S01E01</episode-num>
    <icon src="http://example.com/show.png"/>
  </programme>
</tv>`

	tv, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, tv.Channels, 2)
	require.Len(t, tv.Programmes, 1)

	assert.Equal(t, "channel1", tv.Channels[0].ID)
	assert.Equal(t, "Channel One", tv.Channels[0].DisplayName)
	assert.Equal(t, "http://example.com/icon1.png", tv.Channels[0].Icon)

	assert.Equal(t, "channel2", tv.Channels[1].ID)
	assert.Equal(t, "", tv.Channels[1].Icon)

	prog := tv.Programmes[0]
	assert.Equal(t, "channel1", prog.Channel)
	assert.Equal(t, "Test Show", prog.Title)
	assert.Equal(t, "A test show description", prog.Description)
	assert.Equal(t, "News", prog.Category)
	assert.Equal(t, "S01E01", prog.EpisodeNum)
	assert.Equal(t, "http://example.com/show.png", prog.Icon)
	assert.Equal(t, 2024, prog.Start.Year())
	assert.Equal(t, 12, prog.Start.Hour())
	assert.Equal(t, 13, prog.Stop.Hour())
}

func TestParse_EnrichedFields(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <channel id="ch1">
    <display-name>Test Channel</display-name>
  </channel>
  <programme start="20240315200000 +0000" stop="20240315210000 +0000" channel="ch1">
    <title>Breaking Bad</title>
    <sub-title>Ozymandias</sub-title>
    <desc>Walt faces the consequences.</desc>
    <date>20130915</date>
    <credits>
      <director>Rian Johnson</director>
      <actor>Bryan Cranston</actor>
      <actor>Aaron Paul</actor>
      <writer>Moira Walley-Beckett</writer>
    </credits>
    <category>Drama</category>
    <category>Thriller</category>
    <category>Crime</category>
    <language>en</language>
    <episode-num system="xmltv_ns">4.13.</episode-num>
    <icon src="http://example.com/bb.png"/>
    <rating>
      <value>TV-MA</value>
      <icon src="http://example.com/tvma.png"/>
    </rating>
    <star-rating>
      <value>10/10</value>
    </star-rating>
    <previously-shown />
  </programme>
  <programme start="20240315210000 +0000" stop="20240315220000 +0000" channel="ch1">
    <title>New Show</title>
    <new />
  </programme>
</tv>`

	tv, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, tv.Programmes, 2)

	prog := tv.Programmes[0]
	assert.Equal(t, "Breaking Bad", prog.Title)
	assert.Equal(t, "Ozymandias", prog.Subtitle)
	assert.Equal(t, "20130915", prog.Date)
	assert.Equal(t, "en", prog.Language)
	assert.Equal(t, "4.13.", prog.EpisodeNum)
	assert.Equal(t, "xmltv_ns", prog.EpisodeNumSystem)
	assert.Equal(t, "TV-MA", prog.Rating)
	assert.Equal(t, "http://example.com/tvma.png", prog.RatingIcon)
	assert.Equal(t, "10/10", prog.StarRating)
	assert.True(t, prog.IsPreviouslyShown)
	assert.False(t, prog.IsNew)
	assert.Equal(t, "Drama", prog.Category)
	assert.Contains(t, prog.SubCategories, "Thriller")
	assert.Contains(t, prog.SubCategories, "Crime")
	assert.Contains(t, prog.Credits, "Rian Johnson")
	assert.Contains(t, prog.Credits, "Bryan Cranston")
	assert.Contains(t, prog.Credits, "Aaron Paul")
	assert.Contains(t, prog.Credits, "Moira Walley-Beckett")

	prog2 := tv.Programmes[1]
	assert.Equal(t, "New Show", prog2.Title)
	assert.True(t, prog2.IsNew)
	assert.False(t, prog2.IsPreviouslyShown)
}
