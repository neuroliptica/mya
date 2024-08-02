package main

// Evaluate series of functions before first error.
type Maybe []func() error

func (m Maybe) Eval() error {
	for i := range m {
		err := m[i]()
		if err != nil {
			return err
		}
	}
	return nil
}
