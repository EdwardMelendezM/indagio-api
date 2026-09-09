package worker

import "context"

// PeriodicJob es la interfaz que cualquier Use Case o adaptador debe cumplir
type PeriodicJob interface {
	Name() string
	Execute(ctx context.Context) error
}
