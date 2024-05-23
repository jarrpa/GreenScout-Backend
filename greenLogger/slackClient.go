package greenlogger

import (
	"GreenScoutBackend/constants"

	"github.com/slack-go/slack"
)

var api *slack.Client

var slackAlive = false

func InitSlackAPI(token string) bool {
	if token == "" {
		return false
	}
	api = slack.New(token)

	var validated bool
	if validated = validateToken(); validated {
		slackAlive = true
	}
	return validated
}

func validateToken() bool {
	res, _, err := api.ListTeams(slack.ListTeamsParameters{})

	if err != nil {
		LogError(err, "Problem listing teams the slack bot can access")
		return false
	}

	if len(res) < 1 {
		LogMessage("Slack bot cannot access any workspaces!")
		return false
	}

	return true
}

func ValidateChannelAccess(channel string) bool {
	_, _, err := api.PostMessage(
		channel,
		slack.MsgOptionText("Spinning up server...", false),
		slack.MsgOptionAsUser(true),
	)

	return err == nil
}

func NotifyOnline(online bool) {
	var msg string
	if online {
		msg = "ONLINE"
	} else {
		msg = "OFFLINE"
	}
	_, _, err := api.PostMessage(
		constants.CachedConfigs.SlackConfigs.Channel,
		slack.MsgOptionText("Server status: "+msg, false),
		slack.MsgOptionAsUser(true),
	)

	if err != nil {
		LogError(err, "Problem notifying server of status "+msg)
	}
}

func NotifyMessage(message string) {
	_, _, postErr := api.PostMessage(
		constants.CachedConfigs.SlackConfigs.Channel,
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true),
	)

	if postErr != nil {
		Fatal(postErr, "Problem writing message to slack")
	}

}

func NotifyError(err error, message string) {
	_, _, postErr := api.PostMessage(
		constants.CachedConfigs.SlackConfigs.Channel,
		slack.MsgOptionText("ERR: "+message, false),
		slack.MsgOptionAttachments(slack.Attachment{Text: err.Error()}),
		slack.MsgOptionAsUser(true),
	)

	if postErr != nil {
		Fatal(postErr, "Problem writing error message to slack")
	}

}

func Fatal(err error, message string) {
	api.PostMessage(
		constants.CachedConfigs.SlackConfigs.Channel,
		slack.MsgOptionText("FATAL: "+message, false),
		slack.MsgOptionAttachments(slack.Attachment{Text: err.Error()}),
		slack.MsgOptionAsUser(true),
	)

	api.PostMessage(
		constants.CachedConfigs.SlackConfigs.Channel,
		slack.MsgOptionText("Server status: OFFLINE", false),
		slack.MsgOptionAsUser(true),
	)
}

func NotifyFatal(err error, message string) {
	api.PostMessage(
		constants.CachedConfigs.SlackConfigs.Channel,
		slack.MsgOptionText("FATAL: "+message, false),
		slack.MsgOptionAttachments(slack.Attachment{Text: err.Error()}),
		slack.MsgOptionAsUser(true),
	)
	NotifyOnline(false)
}

func ShutdownSlack() {
	slackAlive = false
}
