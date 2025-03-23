package ui

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/go-color-term/go-color-term/coloring"
	"github.com/muesli/reflow/wordwrap"
	"github.com/vrypan/lemon3/farcaster"
	"github.com/vrypan/lemon3/fctools"
	db "github.com/vrypan/lemon3/localdb"
)

func fidToFname(hub *fctools.FarcasterHub, fid uint64) string {
	db.AssertOpen()
	dbKey := fmt.Sprintf("fid:fname:%d", fid)
	val, err := db.Get([]byte(dbKey))
	if val != nil {
		return string(val)
	}
	msg, err := hub.GetUserData(fid, "USER_DATA_TYPE_DISPLAY")
	if err != nil {
		return "???"
	}
	fname := msg.Data.GetUserDataBody().GetValue()
	if len(fname) == 0 {
		return "???"
	}
	db.Set([]byte(dbKey), []byte(fname))
	return fname
}

func ppTimestamp(ts uint32) string {
	timestamp := time.Unix(int64(ts)+fctools.FARCASTER_EPOCH, 0)
	formattedTime := timestamp.Format("2006-01-02 15:04")
	return coloring.Faint("[" + formattedTime + "]")
}
func PpFname(fname string) string {
	return coloring.Magenta("@" + fname)
}
func ppCastId(fname string, hash []byte) string {
	return PpFname(fname) + coloring.Faint("/"+"0x"+hex.EncodeToString(hash))
}
func ppUrl(url string) string {
	return coloring.Green(url)
}
func addPadding(s string, padding int, paddingString string) string {
	padding_s := strings.Repeat(paddingString, padding)
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i, line := range lines {
		lines[i] = padding_s + line
	}
	return strings.Join(lines, "\n")
}
func boldBlock(s string) string {
	sb := strings.Builder{}
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for _, line := range lines {
		sb.WriteString(coloring.Bold(line))
		sb.WriteString("\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

func FormatCast(hub *fctools.FarcasterHub, msg *farcaster.Message) string {
	body := *msg.Data.GetCastAddBody()
	var builder strings.Builder
	var ptr uint32 = 0
	for i, mention := range body.Mentions {
		builder.WriteString(body.Text[ptr:body.MentionsPositions[i]] + "@" + fidToFname(hub, mention))
		ptr = body.MentionsPositions[i]
	}
	builder.WriteString(body.Text[ptr:])
	textBody := wordwrap.String(builder.String(), 79)

	builder.Reset()
	builder.WriteString(ppCastId(fidToFname(hub, msg.Data.Fid), msg.Hash))
	builder.WriteString(" ")
	builder.WriteString(ppTimestamp(msg.Data.Timestamp))
	builder.WriteString("\n")

	builder.WriteString(textBody)

	if len(body.Embeds) > 0 {
		builder.WriteString("\n----")
	}
	for _, embed := range body.Embeds {
		if embed.GetUrl() != "" {
			builder.WriteString("\n")
			builder.WriteString(ppUrl(embed.GetUrl()))
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
