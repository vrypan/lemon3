package fcclient

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	pb "github.com/vrypan/farcaster-go/farcaster"
	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"
)

func Cast(hubConf HubConfig, username string, key string, text string, enclosureCid string) string {
	var err error
	var privateKey []byte

	if privateKey, err = hex.DecodeString(key[2:]); err != nil {
		log.Fatalf("Private key error: %v\n", err)
	}

	expandedKey := ed25519.NewKeyFromSeed(privateKey)
	publicKey := expandedKey.Public().(ed25519.PublicKey)

	hub := NewFarcasterHub(hubConf)
	defer hub.Close()

	fid, err := hub.GetFidByUsername(username)
	if err != nil {
		log.Fatalf("Unable to get FID for %s: %v\n", username, err)
	}
	var castType pb.CastType
	if len(text) <= 320 {
		castType = pb.CastType(0)
	} else {
		castType = pb.CastType(1)
	}
	embeds := []*pb.Embed{}
	embeds = append(embeds, &pb.Embed{
		Embed: &pb.Embed_Url{Url: "https://lemon3.vrypan.workers.dev/" + enclosureCid},
	})
	embeds = append(embeds, &pb.Embed{
		Embed: &pb.Embed_Url{Url: "lemon3+ipfs://" + enclosureCid},
	})

	messageBody := &pb.CastAddBody{
		Mentions:          nil,
		MentionsPositions: nil,
		Text:              text,
		Type:              castType,
		Embeds:            embeds,
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
	message := CreateMessage(messageData, privateKey, publicKey)
	msg, err := hub.SubmitMessage(message)
	if err != nil {
		log.Fatalf("Error submitting message: %v", err)
	}
	return hex.EncodeToString(msg.Hash)
}

func CreateMessage(messageData *pb.MessageData, signerPrivate []byte, signerPublic []byte) *pb.Message {
	hashScheme := pb.HashScheme(pb.HashScheme_value["HASH_SCHEME_BLAKE3"])
	signatureScheme := pb.SignatureScheme(pb.SignatureScheme_value["SIGNATURE_SCHEME_ED25519"])
	dataBytes, _ := proto.Marshal(messageData)
	signerCombined := append(signerPrivate, signerPublic...)

	hasher := blake3.New()
	hasher.Write(dataBytes)
	hash := hasher.Sum(nil)[:20]

	signature := ed25519.Sign(signerCombined, hash)

	return &pb.Message{
		Data:            messageData,
		Hash:            hash,
		HashScheme:      hashScheme,
		Signature:       signature,
		SignatureScheme: signatureScheme,
		Signer:          signerPublic,
		DataBytes:       dataBytes,
	}
}

func CastGetEmbedUrls(username string, hash string) ([]string, error) {
	var err error
	if !IsInitialized() {
		panic("fcclient.CastGetEmbeds called but hubInstance is not initialized.")
	}
	fid, err := hubInstance.GetFidByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("Unable to get FID for %s: %v\n", username, err)
	}
	hashBytes, err := hex.DecodeString(hash[2:])
	if err != nil {
		return nil, fmt.Errorf("Error parsing hash %s: %v\n", hash, err)
	}
	cast, err := hubInstance.GetCast(fid, hashBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to get cast: %v\n", err)
	}

	embeds := cast.Data.GetCastAddBody().Embeds

	links := []string{}
	for _, e := range embeds {
		if l := e.GetUrl(); l != "" {
			links = append(links, l)
		}
	}
	return links, nil
}

func GetCastsByFname(username string, pageSize uint32, reverse bool) ([]*pb.Message, error) {
	var err error
	if !IsInitialized() {
		panic("fcclient.CastGetEmbeds called but hubInstance is not initialized.")
	}
	fid, err := hubInstance.GetFidByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("Unable to get FID for %s: %v\n", username, err)
	}
	msg, err := hubInstance.GetCastsByFid(fid, pageSize, reverse)

	if err != nil {
		return nil, fmt.Errorf("Failed to get casts for %s: %v\n", username, err)
	}
	return msg.Messages, nil
}
