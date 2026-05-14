package sync

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	scheduleKeyPrefix      = "SCHEDULE_"
	pagerDutyTokenKey      = "PAGERDUTY_TOKEN"
	slackTokenKey          = "SLACK_TOKEN"
	runInterval            = "RUN_INTERVAL_SECONDS"
	pdScheduleLookaheadKey = "PAGERDUTY_SCHEDULE_LOOKAHEAD"
	runIntervalDefault     = 60

	slackCurrentGroupTemplateKey  = "SLACK_CURRENT_GROUP_TEMPLATE"
	slackAllGroupTemplateKey      = "SLACK_ALL_GROUP_TEMPLATE"
	slackAggregateCurrentGroupKey = "SLACK_AGGREGATE_CURRENT_GROUP"

	defaultCurrentGroupTemplate = `current-oncall-{{.Slug}}`
	defaultAllGroupTemplate     = `all-oncall-{{.Slug}}s`
)

// Config is used to configure application
// PagerDutyToken - token used to connect to pagerduty API
// SlackToken - token used to connect to Slack API
type Config struct {
	Schedules                  []Schedule
	PagerDutyToken             string
	SlackToken                 string
	RunIntervalInSeconds       int
	PagerdutyScheduleLookahead time.Duration
	// AggregateCurrentSlackGroup, if non-empty, is an extra Slack user group
	// updated with the union of all schedules' current on-call members.
	AggregateCurrentSlackGroup string
}

// Schedule models a PagerDuty schedule that will be synced with Slack
// ScheduleIDs - All PagerDuty schedule ID's to sync
// AllOnCallGroupName - Slack group name for all members of schedule
// CurrentOnCallGroupName - Slack group name for current person on call
type Schedule struct {
	ScheduleIDs            []string
	AllOnCallGroupName     string
	CurrentOnCallGroupName string
}

// NewConfigFromEnv is a function to generate a config from env varibles
// PAGERDUTY_TOKEN - PagerDuty Token
// SLACK_TOKEN - Slack Token
// SCHEDULE_XXX="id,slug" e.g. 1234,platform-engineer supplies Slug to the Slack group name templates.
//
// SLACK_CURRENT_GROUP_TEMPLATE and SLACK_ALL_GROUP_TEMPLATE are Go text/template strings
// with {{.Slug}} available. Defaults match the historical naming:
// current-oncall-{{.Slug}} and all-oncall-{{.Slug}}s.
//
// SLACK_AGGREGATE_CURRENT_GROUP, when set, names an additional Slack user group filled with
// the union of every schedule's current on-call at sync time (e.g. oncall for @oncall).
func NewConfigFromEnv() (*Config, error) {
	config := &Config{
		PagerDutyToken:             os.Getenv(pagerDutyTokenKey),
		SlackToken:                 os.Getenv(slackTokenKey),
		RunIntervalInSeconds:       runIntervalDefault,
		AggregateCurrentSlackGroup: strings.TrimSpace(os.Getenv(slackAggregateCurrentGroupKey)),
	}

	runInterval := os.Getenv(runInterval)
	v, err := strconv.Atoi(runInterval)
	if err == nil {
		config.RunIntervalInSeconds = v
	}

	pagerdutyScheduleLookahead, err := getPagerdutyScheduleLookahead()
	if err != nil {
		return nil, err
	}
	config.PagerdutyScheduleLookahead = pagerdutyScheduleLookahead

	currentTpl, allTpl, err := loadGroupNameTemplates()
	if err != nil {
		return nil, err
	}

	for _, key := range os.Environ() {
		if strings.HasPrefix(key, scheduleKeyPrefix) {
			value := strings.Split(key, "=")[1]
			scheduleValues := strings.Split(value, ",")
			if len(scheduleValues) != 2 {
				return nil, fmt.Errorf("expecting schedule value to be a comma separated scheduleId,name but got %s", value)
			}

			slug := strings.TrimSpace(scheduleValues[1])
			if slug == "" {
				return nil, fmt.Errorf("schedule slug must not be empty in %s", value)
			}

			currentGroupName, err := renderGroupName(currentTpl, slug)
			if err != nil {
				return nil, fmt.Errorf("schedule %s: %w", value, err)
			}
			allGroupName, err := renderGroupName(allTpl, slug)
			if err != nil {
				return nil, fmt.Errorf("schedule %s: %w", value, err)
			}

			config.Schedules = appendSchedule(config.Schedules, strings.TrimSpace(scheduleValues[0]), currentGroupName, allGroupName)
		}
	}

	if len(config.Schedules) == 0 {
		return nil, fmt.Errorf("expecting at least one schedule defined as an env var using prefix SCHEDULE_")
	}

	return config, nil
}

type groupNameTemplateData struct {
	Slug string
}

func loadGroupNameTemplates() (current *template.Template, all *template.Template, err error) {
	currentStr := strings.TrimSpace(os.Getenv(slackCurrentGroupTemplateKey))
	if currentStr == "" {
		currentStr = defaultCurrentGroupTemplate
	}
	allStr := strings.TrimSpace(os.Getenv(slackAllGroupTemplateKey))
	if allStr == "" {
		allStr = defaultAllGroupTemplate
	}

	current, err = template.New("slack-current-group").Parse(currentStr)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", slackCurrentGroupTemplateKey, err)
	}
	all, err = template.New("slack-all-group").Parse(allStr)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", slackAllGroupTemplateKey, err)
	}
	return current, all, nil
}

func renderGroupName(t *template.Template, slug string) (string, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, groupNameTemplateData{Slug: slug}); err != nil {
		return "", err
	}
	out := strings.TrimSpace(buf.String())
	if out == "" {
		return "", fmt.Errorf("rendered empty Slack group name for slug %q", slug)
	}
	return out, nil
}

func appendSchedule(schedules []Schedule, scheduleID, currentGroupName, allGroupName string) []Schedule {
	newScheduleList := make([]Schedule, len(schedules))
	updated := false

	for i, s := range schedules {
		if s.CurrentOnCallGroupName != currentGroupName {
			newScheduleList[i] = s

			continue
		}

		updated = true

		newScheduleList[i] = Schedule{
			ScheduleIDs:            append(s.ScheduleIDs, scheduleID),
			AllOnCallGroupName:     allGroupName,
			CurrentOnCallGroupName: currentGroupName,
		}
	}

	if !updated {
		newScheduleList = append(newScheduleList, Schedule{
			ScheduleIDs:            []string{scheduleID},
			AllOnCallGroupName:     allGroupName,
			CurrentOnCallGroupName: currentGroupName,
		})
	}

	return newScheduleList
}

func getPagerdutyScheduleLookahead() (time.Duration, error) {
	result := time.Hour * 24 * 100

	pdScheduleLookahead, ok := os.LookupEnv(pdScheduleLookaheadKey)
	if !ok {
		return result, nil
	}

	v, err := time.ParseDuration(pdScheduleLookahead)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s as time.Duration: %w", pdScheduleLookahead, err)
	}

	return v, nil
}
