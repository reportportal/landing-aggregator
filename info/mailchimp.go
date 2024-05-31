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

type MailchimpMember = gochimp3.Member

type MailchimpMemberRequest = gochimp3.MemberRequest

func NewMailchimpClient(apiKey string) *MailchimpClient {
	client := gochimp3.New(apiKey)
	return &MailchimpClient{client}
}

func (client *MailchimpClient) AddSubscription(rq io.Reader, listId string) (*MailchimpMember, error) {
	memberRequest, err := parseMemberRequestBody(rq)
	if err != nil {
		return nil, err
	}

	list, err := client.getList(listId)
	if err != nil {
		return nil, err
	}

	if list.isMemberSubscribed(memberRequest.EmailAddress) {
		return nil, errors.New("email address already subscribed")
	}

	memberRequest.Status = "subscribed"

	response, err := list.AddOrUpdateMember(memberRequest.EmailAddress, memberRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func parseMemberRequestBody(body io.Reader) (*MailchimpMemberRequest, error) {

	var requestBody MailchimpMemberRequest

	bytes, err := io.ReadAll(body)
	if err != nil {
		return nil, errors.New("failed to read request body")
	}

	err = json.Unmarshal(bytes, &requestBody)
	if err != nil {
		return nil, errors.New("invalid request body")
	}

	if requestBody.EmailAddress == "" {
		return nil, errors.New("email address is required")
	}

	if !isValidEmail(requestBody.EmailAddress) {
		return nil, errors.New("invalid email address")
	}

	return &requestBody, nil
}

func (client *MailchimpClient) getList(id string) (*MailchimpList, error) {
	list, err := client.API.GetList(id, nil)
	if err != nil {
		fmt.Printf("Failed to get list '%s'", id)
		return nil, err
	}

	return &MailchimpList{list}, nil
}

func (list *MailchimpList) isMemberSubscribed(id string) bool {
	member, err := list.GetMember(id, &gochimp3.BasicQueryParams{Fields: []string{"status"}})
	if err != nil {
		return false
	}

	return member.Status == "subscribed"
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
