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

type EmailService interface {
	SendEmail(ctx context.Context, sendSmtpEmail SendSmtpEmail) error
	Close(ctx context.Context) error
}

type emailService struct {
	br *brevo.APIClient
}

func NewEmailService(_ context.Context, cfg *config.Config) (EmailService, error) {
	brevoCfg := brevo.NewConfiguration()
	brevoCfg.AddDefaultHeader("api-key", cfg.SMTP.APIKey)

	br := brevo.NewAPIClient(brevoCfg)

	return &emailService{
		br: br,
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
	From            Sender
	To              []Receiver
	Subject         string
	HtmlContent     string
}

func (s *emailService) SendEmail(ctx context.Context, sendSmtpEmail SendSmtpEmail) error {
	if len(sendSmtpEmail.To) > MaxRecipientsPerSend {
		return errors.New("recipients exceeds maximum limit")
	}

	to := make([]brevo.SendSmtpEmailTo, 0)
	for _, r := range sendSmtpEmail.To {
		to = append(to, brevo.SendSmtpEmailTo{
			Email: r.Email,
		})
	}

	body := brevo.SendSmtpEmail{
		Sender: &brevo.SendSmtpEmailSender{
			Email: sendSmtpEmail.From.Email,
		},
		ReplyTo: &brevo.SendSmtpEmailReplyTo{
			Email: sendSmtpEmail.From.Email,
		},
		To:          to,
		Subject:     sendSmtpEmail.Subject,
		HtmlContent: sendSmtpEmail.HtmlContent,
		Tags:        []string{fmt.Sprint(sendSmtpEmail.CampaignEmailID)},
		ScheduledAt: time.Now().Add(5 * time.Second),
	}

	js, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, sendEmailUrl, bytes.NewReader(js))
	if err != nil {
		return err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("api-key", "")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)

	fmt.Println(string(b))

	//_, resp, err := s.br.TransactionalEmailsApi.SendTransacEmail(ctx, brevo.SendSmtpEmail{
	//	Sender: &brevo.SendSmtpEmailSender{
	//		Email: sendSmtpEmail.From.Email,
	//	},
	//	ReplyTo: &brevo.SendSmtpEmailReplyTo{
	//		Email: sendSmtpEmail.From.Email,
	//	},
	//	To:          to,
	//	Subject:     sendSmtpEmail.Subject,
	//	HtmlContent: sendSmtpEmail.HtmlContent,
	//	Tags:        []string{fmt.Sprint(sendSmtpEmail.CampaignEmailID)},
	//})
	//if err != nil {
	//	body, _ := io.ReadAll(resp.Body)
	//	fmt.Println(string(body))
	//	return err
	//}

	return nil
}

func (s *emailService) Close(_ context.Context) error {
	return nil
}

//func (s *emailService) getScheduled() string {
//	currentTime := time.Now()
//
//	// Add 2 minutes
//	twoMinutesLater := time.Now().Add(2 * time.Minute)
//
//	// Format the time in the desired string format
//	formattedTime := time.Now().Add(2 * time.Minute).Format("2006-01-02T15:04:05-07:00")
//
//	// Print the result
//	fmt.Println(formattedTime)
//}
