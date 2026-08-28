package agentloneimpl

// Storer loads a record by id.
type Storer interface { // want `exactly one implementing type in this package \(fileStore\)`
	Load(id string) ([]byte, error)
}

var _ Storer = (*fileStore)(nil)

type fileStore struct{ root string }

// Load reads the record from the configured root.
func (f *fileStore) Load(id string) ([]byte, error) {
	return []byte(f.root + id), nil
}

// Notifier has two implementations, so the count clears it.
type Notifier interface {
	Notify(msg string) error
}

type emailNotifier struct{}

// Notify sends the message by mail.
func (emailNotifier) Notify(msg string) error {
	return nil
}

type smsNotifier struct{}

// Notify sends the message by text.
func (smsNotifier) Notify(msg string) error {
	return nil
}

// Empty carries no method set, so there is nothing to count.
type Empty interface{}

// Unimplemented has no implementation in this package, so the count is zero
// and the analyzer stays quiet rather than guessing about other packages.
type Unimplemented interface {
	Reticulate(splines int) error
}
