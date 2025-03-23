package fctools

import (
	"crypto/ed25519"
	"fmt"
	"time"

	pb "github.com/vrypan/lemon3/farcaster"
	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"
)

func (hub FarcasterHub) SendCast(fid uint64, privateKey []byte, body string, link string) (*pb.MessageData, error) {

	embeds := append([]*pb.Embed{},
		&pb.Embed{
			Embed: &pb.Embed_Url{Url: link},
		},
	)
	messageBody := &pb.CastAddBody{
		Text:   body,
		Type:   pb.CastType_CAST,
		Embeds: embeds,
	}
	messageData := &pb.MessageData{
		Type:      pb.MessageType(pb.MessageType_value["MESSAGE_TYPE_CAST_ADD"]),
		Fid:       fid,
		Timestamp: uint32(time.Now().Unix() - FARCASTER_EPOCH),
		Network:   pb.FarcasterNetwork(pb.FarcasterNetwork_value["FARCASTER_NETWORK_MAINNET"]),
		Body: &pb.MessageData_CastAddBody{
			CastAddBody: messageBody,
		},
	}
	message := CreateMessage(messageData, privateKey)

	_, err := hub.SubmitMessage(message)
	if err != nil {
		return nil, fmt.Errorf("Error submitting message: %v", err)
	}
	// fmt.Printf("Sent: @%d/0x%s\n", msg.Data.Fid, hex.EncodeToString(msg.Hash))
	return messageData, nil
}

func CreateMessage(messageData *pb.MessageData, privateKey []byte) *pb.Message {
	signer := ed25519.NewKeyFromSeed(privateKey)
	publicKey := signer.Public().(ed25519.PublicKey)

	hashScheme := pb.HashScheme(pb.HashScheme_value["HASH_SCHEME_BLAKE3"])
	signatureScheme := pb.SignatureScheme(pb.SignatureScheme_value["SIGNATURE_SCHEME_ED25519"])
	dataBytes, _ := proto.Marshal(messageData)

	hasher := blake3.New()
	hasher.Write(dataBytes)
	hash := hasher.Sum(nil)[:20]

	signature := ed25519.Sign(signer, hash)

	return &pb.Message{
		Data:            messageData,
		Hash:            hash,
		HashScheme:      hashScheme,
		Signature:       signature,
		SignatureScheme: signatureScheme,
		Signer:          publicKey,
		DataBytes:       dataBytes,
	}
}
