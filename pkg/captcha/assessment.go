package captcha

import (
	"context"
	"fmt"

	recaptcha "cloud.google.com/go/recaptchaenterprise/v2/apiv1"
	recaptchapb "cloud.google.com/go/recaptchaenterprise/v2/apiv1/recaptchaenterprisepb"
)

func GetAssessment(projectID, siteKey, token, recaptchaAction string) (*recaptchapb.Assessment, error) {

	ctx := context.Background()

	client, err := recaptcha.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create reCAPTCHA client: %w", err)
	}
	defer client.Close()

	event := &recaptchapb.Event{
		Token:   token,
		SiteKey: siteKey,
	}

	assessment := &recaptchapb.Assessment{
		Event: event,
	}

	request := &recaptchapb.CreateAssessmentRequest{
		Assessment: assessment,
		Parent:     fmt.Sprintf("projects/%s", projectID),
	}

	response, err := client.CreateAssessment(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create reCAPTCHA assessment: %w", err)
	}

	if !response.TokenProperties.Valid {
		return nil, fmt.Errorf("create assessment: invalid token: %v",
			response.TokenProperties.InvalidReason)
	}

	if response.TokenProperties.Action != recaptchaAction {
		return nil, fmt.Errorf("create assessment: action mismatch: got %q want %q", response.TokenProperties.Action, recaptchaAction)
	}

	return response, nil
}
