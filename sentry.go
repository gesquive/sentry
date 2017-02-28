package main

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

// Sentry is the monitoring object
type Sentry struct {
	targets      []SentryTarget
	userAgent    string
	smtpSettings SMTPSettings
	sendAlerts   bool
}

// NewSentry initalizes a new sentry
func NewSentry(targets []SentryTarget, smtpSettings SMTPSettings, version string) *Sentry {
	s := new(Sentry)
	s.targets = targets
	s.smtpSettings = smtpSettings
	s.userAgent = fmt.Sprintf("sentry v%s", version)
	s.sendAlerts = true
	return s
}

// RunCheck runs one check on each target
func (s *Sentry) RunCheck() {
	for i := range s.targets {
		if s.targets[i].NeedsCheck() {
			s.CheckLink(&s.targets[i])
			s.targets[i].ResetRunTime()
		}
	}
}

// Run the monitor
func (s *Sentry) Run() {
	for true {
		for i := range s.targets {
			if s.targets[i].NeedsCheck() {
				go s.CheckLink(&s.targets[i])
				s.targets[i].ResetRunTime()
			}
		}
		time.Sleep(time.Second)
	}
}

// CheckLink tries to check the link status
func (s *Sentry) CheckLink(target *SentryTarget) bool {
	statusCode, err := getHTTPStatus("GET", target.URL, s.userAgent, target.FollowRedirects)
	if err != nil {
		log.Errorf("Error getting http stats of '%s'", target.URL)
	}
	// log.Debugf("target: checked %s, got %d", target.Name, statusCode)
	target.LastReturnCode = statusCode

	// log.Debugf("target: name=%s state=%t", target.Name, target.CurrentState)
	oldState := target.CurrentState
	statusValid := target.IsStatusValid(statusCode)
	if statusValid {
		log.Infof("check: name=%s state=ok status=%d", target.Name, statusCode)
	} else {
		log.Infof("check: name=%s state=err status=%d", target.Name, statusCode)
	}
	target.CurrentState = statusValid
	if oldState != target.CurrentState {
		s.sendStatusAlert(*target)
	}
	return target.CurrentState
}

func getHTTPStatus(method string, url string, userAgent string, followRedirects bool) (int, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !followRedirects {
				return errors.Errorf("Redirects not allowed")
			}
			return nil
		},
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}

func (s *Sentry) sendStatusAlert(target SentryTarget) {
	log.Debugf("Sending alert for %s", target.Name)

	var msg Message
	var statusMsg string
	if target.CurrentState {
		msg.Subject = fmt.Sprintf("[sentry] site online: %s", target.Name)
		statusMsg = fmt.Sprintf("URL is back online: %s", target.URL)
	} else {
		msg.Subject = fmt.Sprintf("[sentry] site offline: %s", target.Name)
		statusMsg = fmt.Sprintf("Received an unexpected return code when requesting URL %s", target.URL)
	}
	msg.ToAddressList = target.AlertEmailList

	msg.TextMessage = fmt.Sprintf(`
Timestamp:  %s
Name:       %s
URL:        %s
StatusCode: %d
-----------------------------------------------------------
%s`,
		time.Now().UTC().Format("Jan 02, 2006 15:04:05 UTC"),
		target.Name, target.URL, target.LastReturnCode, statusMsg)

	if s.sendAlerts {
		sendMessage(msg, s.smtpSettings)
	}

}

// DisableAlerts disables all outgoing alerts
func (s *Sentry) DisableAlerts() {
	s.sendAlerts = false
}
