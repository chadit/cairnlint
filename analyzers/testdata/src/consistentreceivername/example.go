package consistentreceivername

type Server struct{}

// First method sets the receiver name; a later mismatch is flagged.
func (s Server) A() {}
func (srv Server) B() {} // want `receiver name "srv" is inconsistent with "s"`
func (s Server) C() {}

// Not flagged: one consistent name.
type Good struct{}

func (g Good) X() {}
func (g Good) Y() {}
