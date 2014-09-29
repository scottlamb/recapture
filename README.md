Go helper for saving regular expression capture groups.

Modelled after the [RE2 C++
API](https://code.google.com/p/re2/source/browse/re2/re2.h).

Example:

    r := regexp.MustCompile("(.*) (.*) (.*) (.*)")
    var a, b, c, d int
    err := recapture.MatchString(
    	r, "100 40 0100 0x40",
    	recapture.Octal(&a), recapture.Hex(&b),
    	recapture.CRadix(&c), recapture.CRadix(&d))
    if err == nil {
    	// prints 64 64 64 64
    	fmt.Printf("%d %d %d %d", a, b, c, d)
    } else {
    	fmt.Printf("match failed: %v", err)
    }

Get started with:

    go get github.com/scottlamb/recapture
    godoc github.com/scottlamb/recapture
