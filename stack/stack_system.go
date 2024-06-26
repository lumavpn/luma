package stack

import "context"

type systemStack struct {
	handler Handler
}

func NewSystem(options *Options) (Stack, error) {
	system := &systemStack{
		handler: options.Handler,
	}
	return system, nil
}

func (s *systemStack) Stop() error {
	return nil
}

func (s *systemStack) Start(ctx context.Context) error {
	return nil
}
