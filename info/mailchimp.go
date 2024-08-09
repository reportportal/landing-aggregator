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

	if err = list.SetMemberRequestStatus(memberRequest); err != nil {
		return nil, err
	}

	fmt.Printf("Adding member:\n %+v\n", memberRequest)
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

func (l *MailchimpList) SetMemberRequestStatus(rq *MailchimpMemberRequest) error {
	member, err := l.GetMember(rq.EmailAddress, &gochimp3.BasicQueryParams{Fields: []string{"status"}})
	if err != nil {
		if apiErr, ok := err.(*gochimp3.APIError); ok && apiErr.Status == 404 {
			rq.StatusIfNew = "subscribed"
		} else {
			return fmt.Errorf("internal error: %s", err)
		}
	}

	if member.Status == "subscribed" {
		return errors.New("email address already subscribed")
	}

	if rq.Status == "" {
		rq.Status = "subscribed"
	} else if rq.Status != "subscribed" && rq.Status != "pending" {
		return errors.New("invalid status, must be 'subscribed' or 'pending'")
	}

	return nil
}

func (c *MailchimpClient) getList(id string) (*MailchimpList, error) {
	list, err := c.API.GetList(id, nil)
	if err != nil {
		return nil, err
	}

	return &MailchimpList{list}, nil
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
