package main

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/imdario/mergo"
	"github.com/mitchellh/mapstructure"
)

// SentryTarget defines an url to monitor
type SentryTarget struct {
	Name            string   `mapstructure:"name"`
	URL             string   `mapstructure:"url"`
	AlertEmail      string   `mapstructure:"email"`
	AlertEmailList  []string `mapstructure:"emails"`
	CheckInterval   string   `mapstructure:"interval"`
	FollowRedirects bool     `mapstructure:"follow_redirects"`
	ReturnCodes     []int    `mapstructure:"return_codes"`
	interval        time.Duration
	nextCheckTime   time.Time
	LastReturnCode  int
	CurrentState    bool
}

// NewTarget create a sentry target from an interface
func NewTarget(val interface{}) (*SentryTarget, error) {
	t := new(SentryTarget)
	err := mapstructure.Decode(val, t)
	if err != nil {
		return nil, fmt.Errorf("error parsing target: %v", err)
	}

	var targetMap map[string]interface{}
	switch val.(type) {
	case map[string]interface{}:
		targetMap = val.(map[string]interface{})
	case map[interface{}]interface{}:
		targetMap = make(map[string]interface{})
		for key, value := range val.(map[interface{}]interface{}) {
			targetMap[key.(string)] = value
		}
	default:
		return nil, fmt.Errorf("target is an unknown format")
	}

	if v, ok := targetMap["follow_redirects"]; ok {
		t.FollowRedirects = v.(bool)
	} else {
		t.FollowRedirects = true
	}
	if len(t.CheckInterval) > 0 {
		t.interval, err = time.ParseDuration(t.CheckInterval)
		if err != nil {
			return nil, fmt.Errorf("error parsing interval: %v", err)
		}
	}
	t.nextCheckTime = time.Now().UTC()
	t.nextCheckTime = t.nextCheckTime.Add(time.Duration(-1 * t.nextCheckTime.Nanosecond()))
	t.CurrentState = true
	var verifiedEmails []string
	verifiedEmail, err := FormatEmail(t.AlertEmail)
	if err != nil {
		return nil, err
	}
	verifiedEmails = append(verifiedEmails, verifiedEmail)
	for _, email := range t.AlertEmailList {
		verifiedEmail, err := FormatEmail(email)
		if err != nil {
			return nil, err
		}
		verifiedEmails = append(verifiedEmails, verifiedEmail)
	}
	t.AlertEmailList = verifiedEmails
	t.AlertEmail = ""
	return t, nil
}

// SpawnTarget creates a new target with missing fields defaulted to our values
func (t *SentryTarget) SpawnTarget(val interface{}) (*SentryTarget, error) {
	s, err := NewTarget(val)
	if err != nil {
		return nil, err
	}
	err = mergo.Merge(s, t)
	if err != nil {
		return nil, err
	}

	var targetMap map[string]interface{}
	switch val.(type) {
	case map[string]interface{}:
		targetMap = val.(map[string]interface{})
	case map[interface{}]interface{}:
		targetMap = make(map[string]interface{})
		for key, value := range val.(map[interface{}]interface{}) {
			targetMap[key.(string)] = value
		}
	default:
		return nil, fmt.Errorf("target is an unknown format")
	}

	if v, ok := targetMap["follow_redirects"]; ok {
		s.FollowRedirects = v.(bool)
	} else {
		s.FollowRedirects = t.FollowRedirects
	}

	if len(s.CheckInterval) > 0 {
		s.interval, err = time.ParseDuration(s.CheckInterval)
		if err != nil {
			return nil, fmt.Errorf("error parsing interval: %v", err)
		}
	}
	s.nextCheckTime = time.Now().UTC()
	return s, nil
}

// NeedsCheck returns if this target needs to be checked
func (t *SentryTarget) NeedsCheck() bool {
	return time.Now().UTC().After(t.nextCheckTime)
}

// ResetRunTime sets the next run time one interval from Now
func (t *SentryTarget) ResetRunTime() {
	t.nextCheckTime = t.nextCheckTime.Add(t.interval)
	log.Debugf("target: next check for %s is %s", t.Name,
		t.nextCheckTime.Format("Jan 02, 2006 15:04:05 UTC"))
}

// IsStatusValid checks if the given status code is valid
func (t *SentryTarget) IsStatusValid(statusCode int) bool {
	for _, code := range t.ReturnCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}
