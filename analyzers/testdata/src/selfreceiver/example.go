package selfreceiver

type T struct{}

// Flagged: object-oriented receiver names.
func (this *T) Bad()  {} // want `receiver name "this"`
func (self T) Bad2()  {} // want `receiver name "self"`
func (me T) Bad3()    {} // want `receiver name "me"`

// Not flagged: short name derived from the type, or blank.
func (t *T) Good()  {}
func (_ T) Ignored() {}
