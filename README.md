# GreenScout-Backend
The backend/server software for FRC team 1816's scouting app, GreenScout

### IMPORTANT: IT'S FINE THAT SERVER.CRT AND SERVER.KEY ARE PUBLIC. THEY ARE ONLY USED FOR MOCKING HTTPS ON LOCALHOST FOR TESTING
# Getting Started

It's expected that you at least know basic programming principles and practices. Additionally, knowing the [Go Programming Language](https://go.dev/learn/) will be neccessary when contributing to this CLI.

To get started, you'll first need the [Go Programming Language](https://go.dev/dl/), [Git](https://git-scm.com/downloads), and [VS Code (Optional)](https://code.visualstudio.com/Download).

Additional dependencies include [Sqlite 3](https://sqlite.org/download.html), [The Python programming language](https://www.python.org/downloads/), and [pip](https://pypi.org/project/pip/)

Once you have all that installed, open up your terminal and enter this command
```bash
git clone https://github.com/TheGreenMachine/GreenScout-Backend.git
```

This will download the repository onto your computer and to move into it type this
```bash
cd GreenScout-Backend
```

Then, download all of the dependencies of this project with
```bash
go get
```

Then, to run through the setup, run
```bash
go run main.go setup
```

For a thoughrough breakdown of the setup process, go [here](./docs/Setup.md)

Once you've finished that, you can run the server in production. 

If you're on a mac or linux machine, ports 80 and 443 are bound to the root only, so you have to run this with `sudo`
```bash
sudo go run main.go prod
```

If you're running for testing (hosting only on localhost)
```bash
sudo go run main.go test
```

If you want it to write the match numbers of the configured event to the spreadsheet first
```bash
sudo go run main.go prod matches
```

If you are using VS Code, I highly recommend installing the [Golang Extension](https://marketplace.visualstudio.com/items?itemName=golang.Go).

Any additional documentation can be found [here](./docs/). Most documentation can be found in the exhaustively-annotated functions in this project, however.

# Important setup information
  - You will need to know how to port forward in order to ping the server from external networks.
  - You will need a valid domain name, as I could not find a way to get ACME autocert to work without it.

# Roadmap (Things for future developers to add)
  * Admin Features
    * Removing assigned matches from users. (Mutable schedules - talk to Lydia for specifics of what she wants)

  * Discrepencies for multi-scouting - the only one that is implemented right now is average times being too different
  
  * Greenlogger improvement
    * Having the errors also spit out the line of code/method/stacktrace they came from
  

# Contributers

- [Tag C](https://github.com/TagCiccone)