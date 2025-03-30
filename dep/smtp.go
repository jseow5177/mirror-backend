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

const baseUrl = "https://api.brevo.com/v3"

var (
	sendEmailUrl          = fmt.Sprintf("%s/smtp/email", baseUrl)
	createDomainUrl       = fmt.Sprintf("%s/senders/domains", baseUrl)
	authenticateDomainUrl = func(domain string) string {
		return fmt.Sprintf("%s/senders/domains/%s/authenticate", baseUrl, domain)
	}
	getDomainConfigUrl = func(domain string) string {
		return fmt.Sprintf("%s/senders/domains/%s", baseUrl, domain)
	}
	createSenderUrl = fmt.Sprintf("%s/senders", baseUrl)
)

type brevoError struct {
	Message          string `json:"message"`
	DeveloperMessage string `json:"developer_message"`
}

type brevoResp struct {
	Error   *brevoError `json:"error"`
	Message string      `json:"message"`
	Code    string      `json:"code"`
}

type EmailService interface {
	SendEmail(ctx context.Context, sendSmtpEmail *SendSmtpEmail) error
	CreateDomain(ctx context.Context, domain string) (map[string]map[string]interface{}, error)
	CreateSender(ctx context.Context, name, email string) error
	GetDomainConfig(ctx context.Context, domain string) (map[string]map[string]interface{}, error)
	AuthenticateDomain(ctx context.Context, domain string) (bool, error)
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

		if _, err := s.sendHttpRequest(ctx, http.MethodPost, sendEmailUrl, body); err != nil {
			return err
		}
	}

	return nil
}

type createSenderResp struct{}

func (s *emailService) CreateSender(ctx context.Context, name, email string) error {
	b, err := s.sendHttpRequest(ctx, http.MethodPost, createSenderUrl, brevo.CreateSender{
		Name:  name,
		Email: email,
	})
	if err != nil {
		return err
	}

	resp := new(createSenderResp)
	if err := json.Unmarshal(b, resp); err != nil {
		return err
	}

	return nil
}

type createDomainResp struct {
	DnsRecords map[string]map[string]interface{} `json:"dns_records,omitempty"`
}

func (s *emailService) CreateDomain(ctx context.Context, domain string) (map[string]map[string]interface{}, error) {
	b, err := s.sendHttpRequest(ctx, http.MethodPost, createDomainUrl, brevo.CreateDomain{
		Name: domain,
	})
	if err != nil {
		return nil, err
	}

	resp := new(createDomainResp)
	if err := json.Unmarshal(b, resp); err != nil {
		return nil, err
	}

	return resp.DnsRecords, nil
}

type authenticateDomainResp struct{}

func (s *emailService) AuthenticateDomain(ctx context.Context, domain string) (bool, error) {
	b, err := s.sendHttpRequest(ctx, http.MethodPut, authenticateDomainUrl(domain), nil)
	if err != nil {
		return false, err
	}

	resp := new(authenticateDomainResp)
	if err := json.Unmarshal(b, resp); err != nil {
		return false, err
	}

	return true, nil
}

type getDomainConfigResp struct {
	DnsRecords map[string]map[string]interface{} `json:"dns_records,omitempty"`
}

func (s *emailService) GetDomainConfig(ctx context.Context, domain string) (map[string]map[string]interface{}, error) {
	b, err := s.sendHttpRequest(ctx, http.MethodGet, getDomainConfigUrl(domain), nil)
	if err != nil {
		return nil, err
	}

	resp := new(getDomainConfigResp)
	if err := json.Unmarshal(b, resp); err != nil {
		return nil, err
	}

	return resp.DnsRecords, nil
}

func (s *emailService) Close(_ context.Context) error {
	return nil
}

func (s *emailService) sendHttpRequest(_ context.Context, method string, url string, body interface{}) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		js, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewBuffer(js)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("api-key", s.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	brevoResp := new(brevoResp)
	if err := json.Unmarshal(b, brevoResp); err != nil {
		return nil, err
	}

	if brevoResp.Error != nil {
		return nil, fmt.Errorf("encounter brevo error: %s", brevoResp.Error.Message)
	} else if brevoResp.Code != "" {
		return nil, fmt.Errorf("encounter brevo error: %s, code: %s", brevoResp.Message, brevoResp.Code)
	}

	return b, nil
}
