package dep

import (
	"bytes"
	"cdp/config"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	brevo "github.com/getbrevo/brevo-go/lib"
	"io"
	"net/http"
	"time"
)

const (
	MaxRecipientsPerSend = 10
)

var (
	sendEmailUrl = "https://api.brevo.com/v3/smtp/email"
)

type brevoResp struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

type EmailService interface {
	SendEmail(ctx context.Context, sendSmtpEmail *SendSmtpEmail) error
	Close(ctx context.Context) error
}

type emailService struct {
	apiKey string
}

func NewEmailService(_ context.Context, cfg config.Brevo) (EmailService, error) {
	return &emailService{
		apiKey: cfg.APIKey,
	}, nil
}

type Sender struct {
	Email string
	Name  string
}

type Receiver struct {
	Email string
	Name  string
}

type SendSmtpEmail struct {
	CampaignEmailID uint64
	From            *Sender
	To              []*Receiver
	Subject         string
	HtmlContent     string
}

func (s *emailService) SendEmail(ctx context.Context, sendSmtpEmail *SendSmtpEmail) error {
	if len(sendSmtpEmail.To) > MaxRecipientsPerSend {
		return errors.New("recipients exceeds maximum limit")
	}

	for _, r := range sendSmtpEmail.To {
		body := brevo.SendSmtpEmail{
			Sender: &brevo.SendSmtpEmailSender{
				Email: sendSmtpEmail.From.Email,
			},
			ReplyTo: &brevo.SendSmtpEmailReplyTo{
				Email: sendSmtpEmail.From.Email,
			},
			To:          []brevo.SendSmtpEmailTo{{Email: r.Email}},
			Subject:     sendSmtpEmail.Subject,
			HtmlContent: sendSmtpEmail.HtmlContent,
			Tags:        []string{fmt.Sprint(sendSmtpEmail.CampaignEmailID)},
			ScheduledAt: time.Now().Add(10 * time.Second),
		}

		if err := s.postHttpRequest(ctx, sendEmailUrl, body); err != nil {
			return err
		}
	}

	return nil
}

func (s *emailService) Close(_ context.Context) error {
	return nil
}

func (s *emailService) postHttpRequest(_ context.Context, url string, body interface{}) error {
	js, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(js))
	if err != nil {
		return err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("api-key", s.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	brevoResp := new(brevoResp)
	if err := json.Unmarshal(b, brevoResp); err != nil {
		return err
	}

	if brevoResp.Message != "" {
		return fmt.Errorf("encounter brevo error: %s, code: %s", brevoResp.Message, brevoResp.Code)
	}

	return nil
}
