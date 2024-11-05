# HTTP stuff

## handleWithCORS()

The server uses a mostly vanilla version of go's http handler. The biggest exception is the function handleWithCORS(). This exists to get around CORS [(Cross-Origin Resource Sharing)](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS), as the domain of the frontend and backend are different.

## httpResponsef()

This method is an easy wrapper for providing http responses and handling errors. Just provide it with a writer, error message to use if the write fails, and content to send back if the write is successful.

It's also formatted, just like functions such as fmt.Printf()

## writer.WriteHeader()

This method writes the [HTTP Status code](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) it will respond with.

200 is a typical success, 500 is a typical failure.

## Go does concurrency good

HTTP methods run in their own goroutines, so don't worry about clogging up the main thread- go already has really good concurrency for that.
