package json2go

type specialChar string

var (
	comment  = specialChar("?")
	typeName = specialChar("!")
)

var (
	tagName = "json"
)
