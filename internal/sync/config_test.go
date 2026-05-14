package sync

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_NewConfigFromEnv_SingleScheduleDefined(t *testing.T) {
	defer SetEnv("SCHEDULE_PLATFORM", "1234,platform-engineer")()
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("RUN_INTERVAL_SECONDS", "10")()

	config, err := NewConfigFromEnv()

	assert.NoError(t, err)
	assert.Equal(t, "token1", config.PagerDutyToken)
	assert.Equal(t, "secretToken1", config.SlackToken)
	assert.Equal(t, 10, config.RunIntervalInSeconds)
	assert.Equal(t, time.Hour*24*100, config.PagerdutyScheduleLookahead)
	assert.Equal(t, 1, len(config.Schedules))
	assert.Equal(t, "all-oncall-platform-engineers", config.Schedules[0].AllOnCallGroupName)
	assert.Equal(t, "current-oncall-platform-engineer", config.Schedules[0].CurrentOnCallGroupName)

	assert.True(t, assert.ObjectsAreEqualValues([]Schedule{{
		ScheduleIDs:            []string{"1234"},
		AllOnCallGroupName:     "all-oncall-platform-engineers",
		CurrentOnCallGroupName: "current-oncall-platform-engineer",
	}},
		config.Schedules))
}

func Test_NewConfigFromEnv_SingleScheduleDefinedWithDefaultRunInterval(t *testing.T) {
	defer SetEnv("SCHEDULE_PLATFORM", "1234,platform-engineer")()
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()

	config, err := NewConfigFromEnv()

	assert.NoError(t, err)
	assert.Equal(t, "token1", config.PagerDutyToken)
	assert.Equal(t, "secretToken1", config.SlackToken)
	assert.Equal(t, 60, config.RunIntervalInSeconds)
	assert.Equal(t, 1, len(config.Schedules))
	assert.Equal(t, "all-oncall-platform-engineers", config.Schedules[0].AllOnCallGroupName)
	assert.Equal(t, "current-oncall-platform-engineer", config.Schedules[0].CurrentOnCallGroupName)

	assert.True(t, assert.ObjectsAreEqualValues([]Schedule{{
		ScheduleIDs:            []string{"1234"},
		AllOnCallGroupName:     "all-oncall-platform-engineers",
		CurrentOnCallGroupName: "current-oncall-platform-engineer",
	}},
		config.Schedules))
}

func Test_NewConfigFromEnv_SingleScheduleDefinedWithScheduleLookahead(t *testing.T) {
	defer SetEnv("SCHEDULE_PLATFORM", "1234,platform-engineer")()
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("PAGERDUTY_SCHEDULE_LOOKAHEAD", "8760h")()

	config, err := NewConfigFromEnv()

	assert.NoError(t, err)
	assert.Equal(t, "token1", config.PagerDutyToken)
	assert.Equal(t, "secretToken1", config.SlackToken)
	assert.Equal(t, 60, config.RunIntervalInSeconds)
	assert.Equal(t, time.Hour*24*365, config.PagerdutyScheduleLookahead)
	assert.Equal(t, 1, len(config.Schedules))
	assert.Equal(t, "all-oncall-platform-engineers", config.Schedules[0].AllOnCallGroupName)
	assert.Equal(t, "current-oncall-platform-engineer", config.Schedules[0].CurrentOnCallGroupName)

	assert.True(t, assert.ObjectsAreEqualValues([]Schedule{{
		ScheduleIDs:            []string{"1234"},
		AllOnCallGroupName:     "all-oncall-platform-engineers",
		CurrentOnCallGroupName: "current-oncall-platform-engineer",
	}},
		config.Schedules))
}

func Test_NewConfigFromEnv_MultipleScheduleDefined(t *testing.T) {
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("SCHEDULE_PLATFORM", "1234,platform-engineer")()
	defer SetEnv("SCHEDULE_CORE", "abcd,core-engineer")()
	defer SetEnv("SCHEDULE_UK", "efghass,uk-engineer")()

	config, err := NewConfigFromEnv()

	assert.Equal(t, "token1", config.PagerDutyToken)
	assert.Equal(t, "secretToken1", config.SlackToken)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(config.Schedules))

	assert.True(t, assert.ObjectsAreEqualValues([]Schedule{
		{
			ScheduleIDs:            []string{"1234"},
			AllOnCallGroupName:     "all-oncall-platform-engineers",
			CurrentOnCallGroupName: "current-oncall-platform-engineer",
		},
		{
			ScheduleIDs:            []string{"abcd"},
			AllOnCallGroupName:     "all-oncall-core-engineers",
			CurrentOnCallGroupName: "current-oncall-core-engineer",
		},
		{
			ScheduleIDs:            []string{"efghass"},
			AllOnCallGroupName:     "all-oncall-uk-engineers",
			CurrentOnCallGroupName: "current-oncall-uk-engineer",
		},
	},
		config.Schedules))
}

func Test_NewConfigFromEnv_WithScheduleGroups(t *testing.T) {
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("SCHEDULE_CORE_1", "aaaa,core-engineer")()
	defer SetEnv("SCHEDULE_CORE_2", "bbbb,core-engineer")()
	defer SetEnv("SCHEDULE_CORE_3", "cccc,core-engineer")()

	config, err := NewConfigFromEnv()

	assert.Equal(t, "token1", config.PagerDutyToken)
	assert.Equal(t, "secretToken1", config.SlackToken)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(config.Schedules))

	assert.EqualValues(t, []Schedule{
		{
			ScheduleIDs:            []string{"aaaa", "bbbb", "cccc"},
			AllOnCallGroupName:     "all-oncall-core-engineers",
			CurrentOnCallGroupName: "current-oncall-core-engineer",
		},
	},
		config.Schedules)
}

func Test_NewConfigFromEnv_NoSchedulesDefined(t *testing.T) {
	config, err := NewConfigFromEnv()

	assert.Errorf(t, err, "expecting at least one schedule defined as an env var using prefix SCHEDULE_")
	assert.Nil(t, config)
}

func Test_NewConfigFromEnv_InvalidScheduleData(t *testing.T) {
	defer SetEnv("SCHEDULE_PLATFORM", "foo,bar,buzz")()

	config, err := NewConfigFromEnv()

	assert.Errorf(t, err, "expecting schedule value to be a comma separated scheduleId,name but got foo,bar,buzz")
	assert.Nil(t, config)
}

func Test_NewConfigFromEnv_CustomSlackGroupTemplates(t *testing.T) {
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("SLACK_CURRENT_GROUP_TEMPLATE", "{{.Slug}}-oncall")()
	defer SetEnv("SLACK_ALL_GROUP_TEMPLATE", "{{.Slug}}-oncall-all")()
	defer SetEnv("SCHEDULE_US", "pd-us,us")()
	defer SetEnv("SCHEDULE_EU", "pd-eu,eu")()

	config, err := NewConfigFromEnv()

	assert.NoError(t, err)
	assert.Equal(t, 2, len(config.Schedules))
	assert.Empty(t, config.AggregateCurrentSlackGroup)

	got := map[string]Schedule{}
	for _, s := range config.Schedules {
		got[s.CurrentOnCallGroupName] = s
	}
	assert.Equal(t, Schedule{
		ScheduleIDs:            []string{"pd-us"},
		AllOnCallGroupName:     "us-oncall-all",
		CurrentOnCallGroupName: "us-oncall",
	}, got["us-oncall"])
	assert.Equal(t, Schedule{
		ScheduleIDs:            []string{"pd-eu"},
		AllOnCallGroupName:     "eu-oncall-all",
		CurrentOnCallGroupName: "eu-oncall",
	}, got["eu-oncall"])
}

func Test_NewConfigFromEnv_AggregateCurrentSlackGroup(t *testing.T) {
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("SLACK_AGGREGATE_CURRENT_GROUP", "oncall")()
	defer SetEnv("SCHEDULE_PLATFORM", "1234,platform-engineer")()

	config, err := NewConfigFromEnv()

	assert.NoError(t, err)
	assert.Equal(t, "oncall", config.AggregateCurrentSlackGroup)
}

func Test_NewConfigFromEnv_InvalidSlackGroupTemplate(t *testing.T) {
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("SLACK_CURRENT_GROUP_TEMPLATE", "{{.Slug")()
	defer SetEnv("SCHEDULE_PLATFORM", "1234,platform-engineer")()

	config, err := NewConfigFromEnv()

	assert.Nil(t, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), slackCurrentGroupTemplateKey)
}

func Test_NewConfigFromEnv_EmptyScheduleSlug(t *testing.T) {
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("SCHEDULE_PLATFORM", "1234, ")()

	config, err := NewConfigFromEnv()

	assert.Nil(t, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slug must not be empty")
}

func Test_NewConfigFromEnv_ScheduleSlugStripsOuterQuotes(t *testing.T) {
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("SCHEDULE_US", "PJMC036,'us-oncall'")()
	defer SetEnv("SLACK_CURRENT_GROUP_TEMPLATE", "{{.Slug}}")()
	defer SetEnv("SLACK_ALL_GROUP_TEMPLATE", "{{.Slug}}-all")()

	config, err := NewConfigFromEnv()

	assert.NoError(t, err)
	assert.Equal(t, 1, len(config.Schedules))
	assert.Equal(t, "us-oncall", config.Schedules[0].CurrentOnCallGroupName)
	assert.Equal(t, "us-oncall-all", config.Schedules[0].AllOnCallGroupName)
}

func Test_NewConfigFromEnv_AggregateGroupStripsOuterQuotes(t *testing.T) {
	defer SetEnv("PAGERDUTY_TOKEN", "token1")()
	defer SetEnv("SLACK_TOKEN", "secretToken1")()
	defer SetEnv("SCHEDULE_PLATFORM", "1234,platform-engineer")()
	defer SetEnv("SLACK_AGGREGATE_CURRENT_GROUP", "'oncall'")()

	config, err := NewConfigFromEnv()

	assert.NoError(t, err)
	assert.Equal(t, "oncall", config.AggregateCurrentSlackGroup)
}

func SetEnv(key, value string) func() {
	_ = os.Setenv(key, value)
	return func() {
		_ = os.Unsetenv(key)
	}
}
