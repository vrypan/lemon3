package ui

import (
	"strings"

	"github.com/muesli/reflow/wordwrap"
	"github.com/vrypan/lemon3/enclosure"
	"github.com/vrypan/lemon3/farcaster"
	"github.com/vrypan/lemon3/fctools"
)

type CastItem struct {
	Msg       *farcaster.Message
	Rendered  string
	Height    int
	Enclosure *enclosure.Enclosure
}

func NewCastItem(msg *farcaster.Message) *CastItem {
	return &CastItem{
		Msg:      msg,
		Rendered: "",
		Height:   0,
	}
}

func (c *CastItem) Render(hub *fctools.FarcasterHub) *CastItem {
	c.Rendered = c._render(hub)
	c.Height = strings.Count(c.Rendered, "\n")
	return c
}

func (c *CastItem) GetEnclosure(ipfs enclosure.IpfsUploader) error {
	cid := c.Msg.Data.GetCastAddBody().Embeds[0].GetUrl()
	cid = strings.TrimPrefix(cid, "enclosure+ipfs://")
	var err error
	c.Enclosure, err = enclosure.FromCID(ipfs, cid)
	return err
}

func (c *CastItem) _render(hub *fctools.FarcasterHub) string {
	body := c.Msg.Data.GetCastAddBody()
	var builder strings.Builder
	var ptr uint32 = 0
	for i, mention := range body.Mentions {
		builder.WriteString(body.Text[ptr:body.MentionsPositions[i]] + "@" + fidToFname(hub, mention))
		ptr = body.MentionsPositions[i]
	}
	builder.WriteString(body.Text[ptr:])
	textBody := wordwrap.String(builder.String(), 79)

	builder.Reset()
	builder.WriteString(ppCastId(fidToFname(hub, c.Msg.Data.Fid), c.Msg.Hash))
	builder.WriteString(" ")
	builder.WriteString(ppTimestamp(c.Msg.Data.Timestamp))
	builder.WriteString("\n")

	builder.WriteString(textBody)

	if len(body.GetEmbeds()) > 0 && body.GetEmbeds()[0].GetUrl() != "" {
		builder.WriteString("\n")
		if c.Enclosure != nil {
			builder.WriteString(ppUrl(
				c.Enclosure.FileName + " " + c.Enclosure.FileType + " " + c.Enclosure.HumanReadableSize()),
			)
		} else {
			builder.WriteString(ppUrl(body.Embeds[0].GetUrl()))
		}
	}

	out := builder.String()

	builder.Reset()
	for n, l := range strings.Split(out, "\n") {
		prefix := "│ "
		if n == 0 {
			prefix = "┌─ "
		}
		builder.WriteString(prefix + l + "\n")
	}
	builder.WriteString("└───\n")

	out = builder.String()
	return addPadding(out, 0, " ") + "\n"
}
