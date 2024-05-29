package info

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/hanzoai/gochimp3"
)

type MailchimpClient struct {
	*gochimp3.API
}

type MailchimpList struct {
	*gochimp3.ListResponse
}

type MailchimpNewMemberRequest = gochimp3.MemberRequest

type RequestBody struct {
	EmailAddress string `json:"email_address"`
	Status       string `json:"status"`
}

func NewMailchimpClient(apiKey string) *MailchimpClient {
	client := gochimp3.New(apiKey)
	client.User = "landinginfo"
	return &MailchimpClient{client}
}

func (client *MailchimpClient) GetList(id string) (*MailchimpList, error) {
	list, err := client.API.GetList(id, nil)
	if err != nil {
		fmt.Printf("Failed to get list '%s'", id)
		return nil, err
	}

	return &MailchimpList{list}, nil
}

func NewMemberRequest(email string) (*MailchimpNewMemberRequest, error) {
	if !isValidEmail(email) {
		return nil, errors.New("invalid email address")
	}

	newMember := &MailchimpNewMemberRequest{
		EmailAddress: email,
		Status:       "subscribed",
	}

	return newMember, nil
}

func (client *MailchimpClient) AddSubscription(email string, listId string) error {
	list, err := client.GetList(listId)
	if err != nil {
		return err
	}

	newMember, err := NewMemberRequest(email)
	if err != nil {
		return err
	}

	_, err = list.CreateMember(newMember)
	if err != nil {
		fmt.Printf("Failed to add member '%s' to list '%s'", email, listId)
		return err
	}

	return nil
}

func ParseMailchimpRequestBody(body io.Reader) (string, error) {
	var reqBody RequestBody
	bytes, err := io.ReadAll(body)
	if err != nil {
		return "", errors.New("failed to read request body")
	}

	err = json.Unmarshal(bytes, &reqBody)
	if err != nil {
		return "", errors.New("invalid request body")
	}

	return reqBody.EmailAddress, nil
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
