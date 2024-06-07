# A guide to commonly used elements of golang in this project

If you have any more questions, ask Jose. Only contact Tag if you have to. The internet is also a great source of answers.

### I CANNOT STRESS THIS ENOUGH, TAKE [THE TOUR](https://go.dev/tour/list)

# Variables
- vars: variables that can be changes
    - `var example`
    - `var example = "example"`
    - `var example string`
    - `var example string = "example"`
- consts: immutable variable
    - `const example = "example"`
    - `const example string = example"`
- Variables do not need to be statically typed. Providing a type is optional, unlike languages like java.
- There are multiple way to declare variables
    1. Declare a variable beforehand

        ```
        var example {type is optional} {optionally, you could assign it at declaration}
        example = "hi"
        ```
    2. declare it right there

    `example := "example"`

# Packages, functions, and objects

- Like most languages, go organizes files into packages. Each directory holds a package, and instead of accessing the file, you access the methods inside of the files in the directory by accessing the package.
        ```
        package example

        func SayHi() {
            print("hi")
        }

        ---

        package main

        import "GreenScoutBackend/example"

        func main() {
            example.SayHi()
        }

        ```

- Functions are deemed public or private from their capitalization. If a function, variable, struct field, etc. begins with a lowercase letter, it can never leave the scope it originates from (ex. the package for a function). If it begins with an uppercase letter, it can be accessed in any scope. 

- Functions are defined by their parameters and return types
    - You start with the keyword `func`
    - Then, define your method name
    - Then, your parameters
    - Finally, your return type (none if void)
    ```
    func exampleMethod(myParam int) bool {
        return myParam == 1
    }
    ```

- Structs are how go handles objects. They are defined beforehand and then filled out by functions.
    ```
    type Example struct {
        Field1 string
        Field2 int
    }

    func main() {
        ex := Example{
            Field1: "hi"
            Field2: 1
        }

        print(ex.Field1)
    }
    ```
- You can create methods for these structs:
    ```     
    func (e *Example) GetField2 int () {
        return e.Field2;
    }

    func main() {
        ex := Example{
            Field1: "hi"
            Field2: 1
        }

        print(ex.GetField2())
    }
    ```
- Structs can be annotated to support JSON (the same formatting applies to YAML):

    ```
    type Example struct {
        Field1 string `json:"Field 1"`
        Field2 int    `json:"Field 2"`
    }
    ```
        After marshalling to JSON ->

        {
            "Field 1" = "example",
            "Field 2" = 1
        }
    

- Functions can return multiple values.
    ``` 
    func multipleReturns() (int, bool) {
        return 1, true
    }
    ```

- Go forces all variables declared to be used in order to compile. However, if we don't want to use a variable, we can ignore it with an underscore:

    `intResult, _ := multipleReturns()`
# Errors

### Go Forces Error handling. This is great for ensuring clean code pracitices and awareness of possible problems.

- Methods can return errors, which are a type in go. 
- Because go's compiler will not allow compilation if variables are unused, the programmer is forced to handle these errors

    ```
    func myMethod() (int, error) {
        return 1, errors.New("my error")
    }

    func main() {
        exampleVar, err := myMethod()

        if err != nil {
            print("error occurred " + err.Error())
        }
    }
    ```

- There are two ways to see what kind of error you got:

    1. `errors.Is(err, {some predefined error})`
    2. `strings.Contains(err.Error(), "Some part of the error message")`

        ### 1 is always preferable.  2 is sometimes needed.


# Nils, pointers, and seg faults
- Go does not have null. Instead, it has nil. They are similar but different. Importantly, just like a null pointer in java, using a nil where it's not supposed to be will cause a **Segmentation fault** and crash the program. If you know C or C++, you probably just got a little PTSD there.

- Another common cause of crashes is using a NaN where it shouldn't be. To check for this this, you can use `math.isNan()`

- I'm not going to be able to teach pointers here in their entirety, so here are the basics:
    - A pointer, denoted by an `*`, is just the memory address of a given variable. 
    - A dereference, denoted by an `&`, tells the code to access the variable at the pointer, typically used in constructors and such things.
    - Do not try fancy pointer arithmetic. Just don't. 
    - Messing with memory like this is the easiest way to a seg fault. At the same time, forgetting to dereference is the easiest way to spend 3 hours debugging. 
    - All of your code mentors know far more about pointers and memory than I. ASK THEM FOR HELP, I MEAN IT. 

# Commands
- Sometimes, we need to execute commands as though we were in the terminal. There are two ways to do this, with different reasons for doing so.

    First, the common thread:

    `runnable, err := exec.Command("exampleCommand")`

    1. If we just want to execute the comand:

    `runnable.Run()`

    2. If we want the output it spits to the terminal:

    `out, outputErr := runnable.Output()`
     
# Common programming elements
- if statements are very similar to other languages. There are only 2 key differences:
    1. The attached conditions don't have parentheses around them
    2. You can declare variables in-line:
        ```
        if remainder := somevar % 10; remainder > 1 {
            ...
        }
        ```
- for loops are also quite similar. You can declare variables in-line in the exact same way! 
- Additionally, for loops can act as for-each loops: 
    ```
    // You can count at the same time with i, or ignore the index with _

    for i, var in range vars {
    ...
    }
    ```
- Or while loops: 
    ```
    var myBoolean bool = true
    for myBoolean {
    ...
    }
    ```
- Or infinite loops:
    ```
    for {
    ...
    }
    ```

- Recursion: The practice of a function returning itself. Examples are in setup.go

# Strings, slices, and arrays
- Unlike java, you can check string equality with ==

- All arays are also [slices](https://go.dev/tour/moretypes/7). Check out that link for all the fun stuff you can do with them! It comes in very handy.

- Formatting strings is done with many methods, but generally follows this format: `methodf("message %{specifier}", varToGoInSpecifier)`
    * Common specifiers are:
        * %v: any variable
        * %d: number
        * %s: string
        * Full list at https://pkg.go.dev/fmt


- Arrays can be added to using the `append()` function. Even though it's not a function of an array object, you can type {yourarray}.append and it'll automatically surround it as `append(yourarray, {put another slice here})`

- strings.Contains() and slices.Contains() are both incredibly useful.

- strconv.ParseInt() is used to convert a string (like an http header) into an int by specifying the base (always 10) and amount of bits.

- len() gets the length of a slice

- strings.Split() is incredibly helpful in processing strings. Learn it.

- The fmt package is incredibly helpful in many ways. It has tools for input, output, string manipulation, and so much more. Know it!

# Misc
- filepath.Join() is used to create a filepath with platform independent path separators. Always use this for file/directory paths!

- Panicking with `panic()` will send a message to the console and then crash the entire program. `log.Fatal()` will do the same thing, and I'm not sure of the reason for both existing. We will almost never intentionally crash, so avoid these when possible. **It's very important to minimize when these could come up in dependencies, as we can't control that and they could cause the entire server to go down.**


- There are a LOT of code-completion shortcuts offered by go. My favorites include:
    * append
    * forr
    * wr
    * go
    * hand
    * helloWeb
    * hf
    * len

# Data types
- Datatypes are casted like functions: `int(10.1)`
- int division still applies - you'll need to divide as floats to get decimals


- go has a lot of datatypes. these are the common ones:
    - A signed number is one that can be positive or negative
    - An unsigned number is one that can only be positive
    - Numbers can be written MANY ways. I reccomend looking into this - it's good fun!

    * ints: The basic, non-decimal representation of a number
        * They come in 4 signed formats, representing a positive or negative number of so many bits: int8, int16, int32, int64
        * They come in 4 unsigned formats, representing a positive only number of so many bits: uint8, uint16, uint32, uint64
        * the basic `int` is a dynamically allocated 32- or 64-bit signed integer
        * the basic `uint` is a dynamically allocated 32- or 64-bit unsigned integer
    * A `byte` is a uint8
    * floats: The decimal, [floating-point representation](https://www.youtube.com/watch?v=PZRI1IfStY0) of a number
        * They come in 32-bit (float32) and 64-bit (float64) varieties
    * Complex numbers also exist. We don't use them. Unless you guys end up doing really advanced stats. 
    * bools: true or false. These are denoted by the keyword `bool`
    * strings: There is so much to say about strings in this language. The only thing beginners need to know is that the keyword is `string`
    * any: A datatype that can be anything - used when you want to return an uncertain datatype from a function.


- full list [here](https://www.geeksforgeeks.org/data-types-in-go/#)

# More language features
- deferring: This allows a statement to not complete until right before the function scope closes. This is typically used for a file.Close().
    ``` 
    func myFunc() {
        fmt.println("hi")
        defer print("up")
        print("what's ")
    }

    output:
    hi
    what's up
    ```


- You will encounter a lot of **Readers**. These are like streams in other languages - if you read from them once, they don't have any content to be read from again! So, make sure if you want to use the data from a reader multiple times, store it in a variable!

- Go has many ways of dealing with JSON and YAML
    - marshalling/unmarshalling (I've never done this way with yaml): Typically used when directly dealing with the data
        ```
        jsonAsByteArray, err := json.Marshal(someStruct)

        var jsonAsObj ExampleStruct

        json.Unmarshal(jsonAsByteArray, &jsonAsObj)
        ```
    - encoding/decoding: Typically used when dealing with a recipient of the data, like a file or http response
        ```
        var someData ExampleStruct = ...
        yaml.NewEncoder(someWriter).Encode(&someData)

        var data2 ExampleStruct
        yaml.NewDecoder(someReader).Decode(&data2)
        ```

- [Channels](https://gobyexample.com/channels): How goroutines are connected
- goroutines: Lightweight, independent mini-threads that run concurrently. 
    ```
    func printALot(){
        for i := range 100_000_000 {
            print(i)
	    }
    }

    func main() {
        go printALot()
        print("what's up")

        someFuncThatBlocksMainThread()
    }

    printAlot will execute while the main thread is blocked.
    ```
- If you're interested, you can look into tickers, timers, and crons.

# Files and OS
- os.Open() provides a file reference that can only be used for reading
- os.ReadDir() provides information about a directory, such as a list of its files
- os.Exit() crashes the program

- Arrays of bytes are used to represent data in more primitive ways - http request bodies, marshalled JSON, and many other things are byte arrays. Thankfully, these can be easily cast back and forth with strings. 

# SQL
- Sql.Query() is used to query multiple rows
    - If you have multiple rows, use a for loop:

        ```
        rows, err := sql.Query(someStatement)
        for rows.Next() {
            ...
        }
        ```
- Sql.QueryRow() is used to query 1 row of data
    - To scan it, use sql.Row.Scan:

        ```
        response := sql.QueryRow(someStatement)

	    var resultAsVar any

	    scanErr := response.Scan(&resultAsVar)
        ```
- Sql.Exec() is used to execute an sql command without expecting a response

# Inputs

- To get inputs from the terminal, use fmt.Scanln() - examples in setup.go
- To get inputs from the internet, we use an HTTP server on port 443 - it's far too much to convey here, but you'll learn by using it in server.go

# Things to know
- The [curl](https://en.wikipedia.org/wiki/CURL) command in the terminal


