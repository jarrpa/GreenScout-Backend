function GETSCOUTER(match, color, driverStation) {
    if (!Number.isInteger(match) || match < 1 ) {
      return "Please enter a valid match"
    }
  
    var colorAsString = color.toUpperCase();
  
    if (colorAsString != "RED" && colorAsString != "BLUE") {
      return "Please enter a valid color"
    }
  
    if (!Number.isInteger(driverStation) || driverStation < 1 || driverStation > 3) {
      return "Please enter a valid Driverstation"
    }
  
  
  // Make a POST request with form data.
    var resumeBlob = Utilities.newBlob('Hire me!', 'application/json', 'request.json');
  
    var formData =`{"Match":${match}, "isBlue":${colorAsString == "BLUE"}, "DriverStation":${driverStation}}`;
  
  // Because payload is a JavaScript object, it is interpreted as
  // as form data. (No need to specify contentType; it automatically
  // defaults to either 'application/x-www-form-urlencoded'
  // or 'multipart/form-data')
  var options = {
    'method' : 'get',
    'payload' : formData
  };
  
  var response = UrlFetchApp.fetch('https://tagciccone.com/scouterLookup', options);
  return response.getContentText()
  }
  
  const quotes = [
    'Drink water.',
    'Slow down!',
    'Take a breather.',
    'Be nice to venue staff.',
    'Rithwik 2024',
    'Ask your local William Teskey about united states presidents!',
    'Purdy is watching.',
    'Nerd.',
    'It is always funny to mess with Evan.',
    ':)',
    "I couldnt think of any more quotes",
    "No, I will not be telling you every quote I put in here.",
    "Are you cooked or are you cooking?",
    'Remind Aahill to do his webwork',
    'Remind Ethan to do his webwork',
    'Drew Cole 秃头书呆子',
    'Please refrain from bothering Tag about the app',
    'Should you be looking at this, or doing strategy?',
    'Luka Dončić is Devin Booker father',
    'Remind Micheal to clock out.',
    'Be like Usain Bolt wearing heelys.',
    'Do you know where Vihaan is? ',
    'Did you lose the plot, or could it just not keep up with you?',
    '"feet" -Elena',
    'Monster energy is not a substitute for sleeping.',
    'Getting a buzzcut is a good life choice.',
    'Lock in.',
    'Use the :toocool: emote on slack more.',
    'Peace and Love.',
    'What year was the Year Without a Summer?',
    'What year did the second bank of the united states obtain its charter?',
    'Ryan McGoff',
    'Deodorant is a good choice to make.',
    'The Sun is Sunny.',
    "Compartmentalization is healthy if you don't think about it.",
    "At least you're not in the duluth stands.",
    "Knicks on top"
    ]
  
  function GETMOTIVATIONALQUOTE(anything) {
    var quote =  quotes[(Math.floor(Math.random() * quotes.length))]
    return quote;
  }