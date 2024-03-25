from __future__ import print_function

import tbaapiv3client
import sys
from tbaapiv3client.rest import ApiException
from pprint import pprint
import json

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
 
    Matches = {}

    try:
        matchesRaw = api_instance.get_event_matches_simple("2024mndu")
        for match in matchesRaw:
            BlueNumbers = []
            RedNumbers = []
            ex = ""
            for key in match.alliances.blue.team_keys:
                BlueNumbers.append(int(key.strip("frc")))

            for key in match.alliances.red.team_keys:
                RedNumbers.append(int(key.strip("frc")))
            
            Matches.update({match.match_number: {"Blue":BlueNumbers, "Red": RedNumbers}})                        

    except ApiException as e:
        print("Exception when calling EventApi->get_event_teams: %s\n" % e)

    json = json.dumps(Matches)
    file = open("schedule/schedule.json", "w")
    file.write(json)

    print("Finished Filling Out Match schedule!")
