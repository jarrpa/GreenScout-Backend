# User Data

Users contain quite a lot of information. I'm going to list a few important ones here, leaving the rest as an exercise in documentation for future devs.

Username: The username, lowercase only. You can theoretically get an uppercase one by logging in with the CLI, but then you can never log into the app with it. Use this for any lookups that require input that's human-enterable, such as from the CLI.

UUID: The unique, primary key identifier. Use this for lookups that don't need to be human-readable, such as internal ones in the backend code.

pfp: The relative path of the pfp image, assuming a cwd of ./pfp/pictures/

score: The amount of matches scouted at the current event

life score: The amount of matches scouted of all time

high score: The most matches scouted by a given user at any single event