package pulsesms

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// MessageID is the internal ID of a Pulse SMS message
type MessageID = int

// DeviceID is the generated internal ID of the device used to interact with a PulseSMS account
type DeviceID = int

type Message struct {
	ID             MessageID      `json:"id,omitempty"`
	ConversationID ConversationID `json:"conversation_id,omitempty"`
	DeviceID       DeviceID       `json:"device_id,omitempty"`
	Type           int            `json:"type,omitempty"`
	Data           string         `json:"data,omitempty"`
	Timestamp      int64          `json:"timestamp,omitempty"`
	MimeType       string         `json:"mime_type,omitempty"`
	Read           bool           `json:"read,omitempty"`
	Seen           bool           `json:"seen,omitempty"`
	From           string         `json:"from,omitempty"`
	Archive        bool           `json:"archive,omitempty"`
	SentDevice     DeviceID       `json:"sent_device,omitempty"`
	SimStamp       string         `json:"sim_stamp,omitempty"`
	Snippet        string         `json:"snippet,omitempty"`
}

type sendMessageRequest struct {
	AccountID            AccountID `json:"account_id,omitempty"`
	Data                 string    `json:"data,omitempty"`
	DeviceConversationID int       `json:"device_conversation_id,omitempty"`
	DeviceID             DeviceID  `json:"device_id,omitempty"`
	MessageType          int       `json:"message_type,omitempty"`
	MimeType             string    `json:"mime_type,omitempty"`
	Read                 bool      `json:"read,omitempty"`
	Seen                 bool      `json:"seen,omitempty"`
	SentDevice           DeviceID  `json:"sent_device"`
	Timestamp            int64     `json:"timestamp,omitempty"`
}

type updateConversationRequest struct {
	AccountID AccountID `json:"account_id,omitempty"`
	Read      bool      `json:"read,omitempty"`
	Timestamp int64     `json:"timestamp,omitempty"`
	Snippet   string    `json:"snippet,omitempty"`
}

func generateID() int {
	const min = 1
	const max = 922337203685477

	s := rand.Float64()
	x := s * (max - min + 1)

	return int(math.Floor(x) + min)
}

func (c *Client) GetMessages(conversationID int, offset int) ([]Message, error) {
	msgs := []Message{}
	const limit = 70

	endpoint := c.getUrl(EndpointMessages)

	resp, err := c.api.R().
		SetQueryParam("account_id", fmt.Sprint(c.accountID)).
		SetQueryParam("conversation_id", fmt.Sprint(conversationID)).
		SetQueryParam("offset", fmt.Sprint(offset)).
		SetQueryParam("limit", fmt.Sprint(limit)).
		SetResult(&msgs).
		Get(endpoint)

	if resp.StatusCode() > 200 || err != nil {
		fmt.Printf("%v: %s\n", resp.StatusCode(), resp.Status())
		return nil, err
	}

	result := []Message{}
	for _, m := range msgs {
		err := decryptMessage(c.crypto.cipher, &m)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt message %v", err)
		}
		result = append(result, m)

	}

	return result, nil

}

func (c *Client) SendMessage(data string, conversationID int) error {
	id := generateID()

	// TODO accept mimetype
	snippet := fmt.Sprintf("You: %s", data)

	mimetype, err := encrypt(c.crypto.cipher, "text/plain")
	if err != nil {
		return err
	}
	encData, err := encrypt(c.crypto.cipher, data)
	if err != nil {
		return err
	}
	encSnippet, err := encrypt(c.crypto.cipher, snippet)
	if err != nil {
		return err
	}

	timestamp := time.Now().Unix()

	req := sendMessageRequest{
		AccountID:            c.accountID,
		Data:                 encData,
		DeviceConversationID: conversationID,
		DeviceID:             id,
		MessageType:          2,
		Timestamp:            timestamp,
		MimeType:             mimetype,
		Read:                 false,
		Seen:                 false,
		SentDevice:           1,
	}

	endpoint := c.getUrl(EndpointAddMessage)
	resp, err := c.api.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		Post(endpoint)

	if resp.StatusCode() > 200 || err != nil {
		fmt.Printf("%v: %s\n", resp.StatusCode(), resp.Status())
		return err
	}
	fmt.Println("sent message")
	fmt.Println(resp.StatusCode(), resp.Status())
	fmt.Println(string(resp.Body()))

	err = c.updateConversation(conversationID, encSnippet, timestamp)
	if err != nil {
		return err
	}

	return nil

}
