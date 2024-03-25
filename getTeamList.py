from __future__ import print_function

import tbaapiv3client
import sys
import os
from tbaapiv3client.rest import ApiException
from pprint import pprint

# Define host
configuration = tbaapiv3client.Configuration(
    host = "https://www.thebluealliance.com/api/v3"
)

# Api ket 
configuration = tbaapiv3client.Configuration(
    host = "https://www.thebluealliance.com/api/v3",
    api_key = {
        'X-TBA-Auth-Key': '' #Put your auth key here
    }
)


# Enter context with api client
with tbaapiv3client.ApiClient(configuration) as api_client:
    api_instance = tbaapiv3client.EventApi(api_client)

    event_key = sys.argv[1] # Arg is event name
    filePath = f"TeamLists/${event_key}"
 
    file = open(filePath, 'w')

    try:
        event_name = api_instance.get_event(event_key)
        file.write(event_name.short_name + "\n")
        
        api_response = api_instance.get_event_teams_simple(event_key)
        teamNumbers = []
        for team in api_response:
            teamNumbers.append(team.team_number)
        
        for number in sorted(teamNumbers):
            file.write(str(number) + "\n")

    except ApiException as e:
        print("Exception when calling EventApi->get_event_teams: %s\n" % e)

    print("Finished filling out team list")
