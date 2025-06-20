package lemon3libs

import (
	"encoding/json"
	"fmt"
	"strings"

	pb "github.com/vrypan/farcaster-go/farcaster"
)

const FARCASTER_EPOCH int64 = 1609459200

type L3Cast struct {
	Fid        uint64
	Fname      string
	Timestamp  uint64
	Hash       string //0x-prefixed
	Lemon3Cid  string
	Text       string
	Lemon3Data *Lemon3Metadata
}

func FromPbMessage(msg *pb.Message) (*L3Cast, error) {
	l3c := L3Cast{}
	cid := l3CidFromCast(msg.Data.GetCastAddBody())
	if cid == "" {
		return nil, nil
	}
	l3c.Lemon3Cid = cid
	l3c.Fid = msg.Data.Fid
	l3c.Timestamp = uint64(msg.Data.Timestamp) + uint64(FARCASTER_EPOCH)
	l3c.Hash = fmt.Sprintf("0x%x", msg.Hash)
	l3c.Text = msg.Data.GetCastAddBody().Text

	var err error
	l3c.Lemon3Data, err = FromCid(cid)
	if err != nil {
		return nil, err
	}
	return &l3c, nil
}

func (c *L3Cast) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func FromJSON(input string) (*L3Cast, error) {
	var cast L3Cast
	err := json.Unmarshal([]byte(input), &cast)
	if err != nil {
		return nil, err
	}
	return &cast, nil
}

func l3CidFromCast(cast *pb.CastAddBody) string {
	embeds := cast.Embeds
	for _, e := range embeds {
		if l := e.GetUrl(); l != "" {
			if strings.HasPrefix(l, "lemon3+ipfs://") {
				return strings.TrimPrefix(l, "lemon3+ipfs://")
			}
		}
	}
	return ""
}
