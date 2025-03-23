package fctools

import (
	"context"
	"fmt"
	"log"

	//"time"
	"crypto/ed25519"
	"encoding/json"

	pb "github.com/vrypan/lemon3/farcaster"
	"github.com/zeebo/blake3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const FARCASTER_EPOCH int64 = 1609459200

type FarcasterHub struct {
	hubAddr    string
	conn       *grpc.ClientConn
	client     pb.HubServiceClient
	ctx        context.Context
	ctx_cancel context.CancelFunc
}

func (f *FarcasterHub) Client() pb.HubServiceClient {
	return f.client
}
func NewFarcasterHub(
	hubAddress string,
	useSsl bool,
) *FarcasterHub {
	cred := insecure.NewCredentials()

	if useSsl {
		cred = credentials.NewClientTLSFromCert(nil, "")
	}

	conn, err := grpc.DialContext(context.Background(), hubAddress, grpc.WithTransportCredentials(cred))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	client := pb.NewHubServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	//ctx = metadata.AppendToOutgoingContext(ctx, "x-api-key", "NEYNAR_API_DOCS_BAD")
	return &FarcasterHub{
		hubAddr:    hubAddress,
		conn:       conn,
		client:     client,
		ctx:        ctx,
		ctx_cancel: cancel,
	}
}

func (h FarcasterHub) Close() {
	h.conn.Close()
	h.ctx_cancel()
}

func (hub FarcasterHub) HubInfo() ([]byte, error) {
	res, err := hub.client.GetInfo(hub.ctx, &pb.HubInfoRequest{DbStats: false})
	if err != nil {
		log.Fatalf("could not get HubInfo: %v", err)
		return nil, err
	}
	b, err := json.Marshal(res)
	return b, err
}

func (hub FarcasterHub) SubmitMessageData(messageData *pb.MessageData, signerPrivate, signerPublic []byte) (*pb.Message, error) {
	const hashLen = 20
	dataBytes, err := proto.Marshal(messageData)
	if err != nil {
		return nil, err
	}
	fullHash := blake3.Sum256(dataBytes)
	hash := fullHash[:hashLen]
	signature := ed25519.Sign(append(signerPrivate, signerPublic...), hash)
	message := pb.Message{
		Data:            messageData,
		Hash:            hash,
		HashScheme:      pb.HashScheme_HASH_SCHEME_BLAKE3,
		Signature:       signature,
		SignatureScheme: pb.SignatureScheme_SIGNATURE_SCHEME_ED25519,
		Signer:          signerPublic,
		DataBytes:       dataBytes,
	}
	return hub.client.SubmitMessage(hub.ctx, &message)
}

func (hub FarcasterHub) SubmitMessage(message *pb.Message) (*pb.Message, error) {
	msg, err := hub.client.SubmitMessage(hub.ctx, message)
	return msg, err
}

func (hub FarcasterHub) GetUserData(fid uint64, user_data_type string) (*pb.Message, error) {
	udt := pb.UserDataType(pb.UserDataType_value[user_data_type])
	message, err := hub.client.GetUserData(hub.ctx, &pb.UserDataRequest{Fid: fid, UserDataType: udt})
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (hub FarcasterHub) GetUsernameProofsByFid(fid uint64) (*pb.UsernameProofsResponse, error) {
	msg, err := hub.client.GetUserNameProofsByFid(hub.ctx, &pb.FidRequest{Fid: fid})
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (hub FarcasterHub) GetFidByUsername(username string) (uint64, error) {
	message, err := hub.client.GetUsernameProof(hub.ctx, &pb.UsernameProofRequest{Name: []byte(username)})
	if err != nil {
		return 0, fmt.Errorf("failed to get username proof: %w", err)
	}
	return message.Fid, nil
}

func (hub FarcasterHub) GetCastsByFid(fid uint64, start []byte, pageSize uint32) (*pb.MessagesResponse, error) {
	reverse := true
	msg, err := hub.client.GetCastsByFid(hub.ctx, &pb.FidRequest{Fid: fid, Reverse: &reverse, PageSize: &pageSize, PageToken: start})
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (hub FarcasterHub) GetReactionsByFid(fid uint64, start []byte, pageSize uint32) (*pb.MessagesResponse, error) {
	reverse := true
	msg, err := hub.client.GetReactionsByFid(hub.ctx,
		&pb.ReactionsByFidRequest{Fid: fid, Reverse: &reverse, PageSize: &pageSize, PageToken: start},
	)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
func (hub FarcasterHub) GetLinksByFid(fid uint64, start []byte, pageSize uint32) (*pb.MessagesResponse, error) {
	reverse := true
	msg, err := hub.client.GetLinksByFid(hub.ctx, &pb.LinksByFidRequest{
		Fid:       fid,
		Reverse:   &reverse,
		PageSize:  &pageSize,
		PageToken: start,
	})
	//msg, err := hub.client.GetCastsByFid(hub.ctx, &pb.FidRequest{Fid: fid, Reverse: &reverse, PageSize: &pageSize, PageToken: start})
	if err != nil {
		return nil, err
	}
	return msg, nil
}
func (hub FarcasterHub) GetCast(fid uint64, hash []byte) (*pb.Message, error) {
	return hub.client.GetCast(hub.ctx, &pb.CastId{Fid: fid, Hash: hash})
}

func (hub FarcasterHub) GetCastReplies(fid uint64, hash []byte) (*pb.MessagesResponse, error) {
	return hub.client.GetCastsByParent(
		hub.ctx,
		&pb.CastsByParentRequest{
			Parent: &pb.CastsByParentRequest_ParentCastId{
				ParentCastId: &pb.CastId{Fid: fid, Hash: hash},
			},
		},
	)
}

func ResignMessage(message *pb.Message, signerPrivate []byte) *pb.Message {
	if message.SignatureScheme != pb.SignatureScheme(pb.SignatureScheme_value["SIGNATURE_SCHEME_ED25519"]) {
		fmt.Println("Not ED25519???")
		return message
	}
	signer := ed25519.NewKeyFromSeed(signerPrivate)
	hash := message.Hash
	signature := ed25519.Sign(signer, hash)
	message.Signature = signature
	message.Signer = signer.Public().(ed25519.PublicKey)

	return message
}
func (hub FarcasterHub) Subscribe(fid uint64) (grpc.ServerStreamingClient[pb.HubEvent], error) {
	return hub.client.Subscribe(hub.ctx, &pb.SubscribeRequest{})
}

func DefaultMetadataKeyInterceptor(key, value string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Inject metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		md.Append(key, value)

		// Create new context with metadata
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Proceed with the request
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
