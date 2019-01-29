package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/smtp"
	"regexp"
	"time"

	"github.com/jordan-wright/email"
	"github.com/knadh/otpgateway"
)

const (
	providerID  = "smtp"
	channelName = "E-mail"
	maxOTPlen   = 6
)

// http://www.golangprograms.com/regular-expression-to-validate-email-address.html
var reMail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// cfg represents an SMTP server's credentials.
type cfg struct {
	Host         string `json:"Host"`
	Port         int    `json:"Port"`
	AuthProtocol string `json:"AuthProtocol"`
	User         string `json:"User"`
	Password     string `json:"Password"`
	FromEmail    string `json:"FromEmail"`
	SendTimeout  int    `json:"SendTimeout"`
	MaxConns     int    `json:"MaxConns"`
}

type emailer struct {
	cfg     cfg
	timeout time.Duration
	mailer  *email.Pool
}

// New creates and returns an e-mail Provider backend.
func New(jsonCfg []byte) (otpgateway.Provider, error) {
	var c cfg
	if err := json.Unmarshal(jsonCfg, &c); err != nil {
		return nil, fmt.Errorf("error reading config: %v", err)
	}

	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 25
	}
	if c.MaxConns == 0 {
		c.MaxConns = 1
	}
	if c.FromEmail == "" {
		c.FromEmail = "otp@localhost"
	}

	// Initialize the SMTP mailer.
	var auth smtp.Auth
	if c.AuthProtocol == "cram" {
		auth = smtp.CRAMMD5Auth(c.User, c.Password)
	} else {
		auth = smtp.PlainAuth("", c.User, c.Password, c.Host)
	}

	pool, err := email.NewPool(fmt.Sprintf("%s:%d", c.Host, c.Port), c.MaxConns, auth)
	if err != nil {
		return nil, err
	}

	// Push timeout.
	t := 5
	if c.SendTimeout == 0 {
		t = c.SendTimeout
	}

	return &emailer{
		mailer:  pool,
		cfg:     c,
		timeout: time.Second * time.Duration(t),
	}, nil
}

// ID returns the Provider's ID.
func (e *emailer) ID() string {
	return providerID
}

// ChannelName returns the e-mail Provider's name.
func (e *emailer) ChannelName() string {
	return channelName
}

// Description returns help text for the e-mail verification Provider.
func (e *emailer) Description() string {
	return fmt.Sprintf(`
		We've e-mailed you a %d digit code.
		Please check your e-mail and enter the code here
		to complete the verification.`, maxOTPlen)
}

// ValidateAddress "validates" an e-mail address.
func (e *emailer) ValidateAddress(to string) error {
	if !reMail.MatchString(to) {
		return errors.New("invalid e-mail address")
	}
	return nil
}

// Push pushes an e-mail to the SMTP server.
func (e *emailer) Push(toAddr string, subject string, m []byte) error {
	return e.mailer.Send(&email.Email{
		From:    e.cfg.FromEmail,
		To:      []string{toAddr},
		Subject: subject,
		HTML:    m,
	}, e.timeout)
}

// MaxOTPLen returns the maximum allowed length of the OTP value.
func (e *emailer) MaxOTPLen() int {
	return maxOTPlen
}

// MaxBodyLen returns the max permitted body size.
func (e *emailer) MaxBodyLen() int {
	return 100 * 1024
}
