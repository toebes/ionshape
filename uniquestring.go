package main

import "strings"

type contextCount struct {
	context string
	count   int
}
type uniqueString map[string]contextCount

// set allows adding a context string
// Leading/trailing blanks are tossed and blank strings are ignored
//
func (u uniqueString) set(val string, context string) {

	valStr := strings.TrimSpace(strings.ReplaceAll(val, "\n", "\\n"))
	if len(valStr) > 0 {
		oldval, found := u[valStr]
		if !found {
			// Not been used before, so just add it to the context
			u[valStr] = contextCount{context: context, count: 1}
		} else {
			// We saw it before, so increment the count and add us to the list of contexts for it
			u[valStr] = contextCount{context: oldval.context + "," + context, count: oldval.count + 1}
		}
	}
}

// get() returns a proper context string
// If there were no references, the string will be blank
// If there was exactly one, then we return the actual string
// If there was more than one then we want to know all the disagreements.
// For example, if we were given:
//          val        context
//          -------    ---------
//          goBILDA    document
//          goBILDA    part
// Then the result would be "goBILDA"
//
//          val        context
//          -------    ---------
//          goBILDA    document
//          GoBilda    part
// Would generate "goBILDA/document ALSO[GoBilda/part]"
//
//          val        context
//          -------    ---------
//          goBILDA    document
//          goBILDA    part
//          GoBilda    part
// Would generate "goBILDA/document,part ALSO[GoBilda/part]"
//          val        context
//          -------    ---------
//          goBILDA    document
//          GoBilda    part
//          GoBilda    part
// Would generate "goBILDA/document ALSO[GoBilda/part,part]"
//
//          val        context
//          -------    ---------
//          goBILDA    document
//          GoBilda    part
//          GoBilda    part
//          GOBILDA    assembly
// Would generate "GoBILDA/part,part ALSO:[goBILDA/document GOBILDA/assembly]"

func (u uniqueString) get() string {
	result := ""
	keycount := 0
	bestmatch := ""
	most := 0
	// Find the most common item
	for key, item := range u {
		// Keep track of how many keys there were so we know if we will have any extra context
		keycount++
		// Is this more common than what we found (we start out with nothing)
		if item.count > most {
			// Yes, so save away our result and the match information for more searching
			result = key
			bestmatch = key
			most = item.count
		}
	}
	// If there was more than one then we need to append the extra information
	if keycount > 1 {
		extra := "/" + u[bestmatch].context + " ALSO:["
		// Go through all the keys
		for key, item := range u {
			// and ignore the one that we already found
			if key != bestmatch {
				// Append the extra context information to what we prepared
				result += extra + key + ":" + item.context
				extra = " "
			}
		}
		result += "]"
	}
	return result
}
