package slack

import (
	greenlogger "GreenScoutBackend/greenLogger"

	"github.com/slack-go/slack"
)

var api *slack.Client

func InitSlackAPI(token string) bool {
	api = slack.New(token)
	return validateToken()
}

func validateToken() bool {
	res, _, err := api.ListTeams(slack.ListTeamsParameters{})

	if err != nil {
		greenlogger.LogError(err, "Problem listing teams the slack bot can access")
		return false
	}

	if len(res) < 1 {
		greenlogger.LogMessage("Slack bot cannot access any workspaces!")
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

	return err != nil
}
