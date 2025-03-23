package ui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vrypan/lemon3/farcaster"
)

func (m *MainModel) SubscribeToFcEvents() tea.Cmd {
	stream, err := m.Hub.Subscribe(1)
	if err != nil {
		return func() tea.Msg {
			return SystemError{Error: err}
		}
	}

	go func() {
		defer close(m.chErrs)
		defer close(m.chFcMsg)
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					m.chErrs <- fmt.Errorf("error receiving from stream: %w", err)
				}
				return
			}
			if resp.GetType() != farcaster.HubEventType_HUB_EVENT_TYPE_MERGE_MESSAGE {
				continue
			}
			msgData := resp.GetMergeMessageBody().GetMessage().GetData()
			switch msgData.GetType() {
			case farcaster.MessageType_MESSAGE_TYPE_CAST_ADD:
				cast := msgData.GetCastAddBody()
				for _, e := range cast.GetEmbeds() {
					if !strings.HasPrefix(e.GetUrl(), "enclosure+ipfs://") {
						continue
					}
					m.chFcMsg <- resp.GetMergeMessageBody().GetMessage()
					break
				}
			case farcaster.MessageType_MESSAGE_TYPE_LINK_ADD:
				if msgData.Fid != m.Fid {
					continue
				}
				m.chFcMsg <- resp.GetMergeMessageBody().GetMessage()
			case farcaster.MessageType_MESSAGE_TYPE_LINK_REMOVE:
				if msgData.Fid != m.Fid {
					continue
				}
				m.chFcMsg <- resp.GetMergeMessageBody().GetMessage()
			}
		}
	}()
	return nil
}

func (m *MainModel) InitFollows() {
	msg, err := m.Hub.GetLinksByFid(m.Fid, nil, 2000)
	if err != nil {
		fmt.Println("Error getting follows:", err)
		return
	}
	for _, link := range msg.Messages {
		m.chFcMsg <- link
	}
}
