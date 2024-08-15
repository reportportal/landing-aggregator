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

type MailchimpMemberStatus string

const (
	MailchimpMemberSubscribed   MailchimpMemberStatus = "subscribed"
	MailchimpMemberUnsubscribed MailchimpMemberStatus = "unsubscribed"
	MailchimpMemberPending      MailchimpMemberStatus = "pending"
	MailchimpMemberNone         MailchimpMemberStatus = ""
)

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

	status, err := list.getMemberStatus(memberRequest)
	if err != nil {
		return nil, err
	}

	switch status {
	case MailchimpMemberSubscribed:
		return nil, errors.New("email address already subscribed")
	case MailchimpMemberPending:
		return nil, errors.New("email address already pending")
	case MailchimpMemberUnsubscribed:
	case MailchimpMemberNone:
		memberRequest.StatusIfNew = memberRequest.Status
	default:
		return nil, errors.New("internal error")
	}

	response, err := list.AddOrUpdateMember(memberRequest.EmailAddress, memberRequest)
	if err != nil {
		if apiErr, ok := err.(*gochimp3.APIError); ok {
			if apiErr.Title == "Member In Compliance State" {
				return list.ResubsribeMember(memberRequest.EmailAddress, memberRequest)
			}
			return nil, apiErr
		}
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

	if requestBody.Status == "" {
		requestBody.Status = "subscribed"
	}

	if requestBody.Status != "subscribed" && requestBody.Status != "pending" {
		return nil, errors.New("invalid status, must be 'subscribed' or 'pending'")
	}

	return &requestBody, nil
}

func (l *MailchimpList) getMemberStatus(rq *MailchimpMemberRequest) (MailchimpMemberStatus, error) {
	member, err := l.GetMember(rq.EmailAddress, &gochimp3.BasicQueryParams{Fields: []string{"status"}})
	if err != nil {
		if apiErr, ok := err.(*gochimp3.APIError); ok && apiErr.Status == 404 {
			return MailchimpMemberNone, nil
		} else {
			return "", fmt.Errorf("internal error: %s", err)
		}
	}

	return MailchimpMemberStatus(member.Status), nil
}

func (l *MailchimpList) ResubsribeMember(email string, rq *MailchimpMemberRequest) (*MailchimpMember, error) {
	rq.Status = "pending"
	return l.UpdateMember(email, rq)
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

func CanMakeMailchimpRequest(client *MailchimpClient) error {
	if client == nil {
		return errors.New("mailchimp client is not initialized")
	}
	return nil
}
