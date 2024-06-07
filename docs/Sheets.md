# Google Sheets integration

This project heavily relies on the dumpster fire that is the Google Sheets API. Good luck. 

https://developers.google.com/sheets/api/quickstart/go

## THE MOST IMPORTANT FUNCTION: writeTeamDataToLine()
This function, as it says, writes the data from one scouting entry to a line. This is THE season-specific method. Every entry in the interface is another cell in the specified row, so edit the position and content of those to alter what gets written to the spreadsheet. 