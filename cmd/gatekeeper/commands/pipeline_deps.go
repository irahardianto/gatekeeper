package commands

import (
	"context"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/formatter"
	"github.com/irahardianto/gatekeeper/internal/engine/gate"
)

// DockerChecker abstracts Docker pre-flight checks.
type DockerChecker interface {
	CheckDocker(ctx context.Context) error
}

// GateCreator abstracts the creation of gate instances from configuration.
type GateCreator interface {
	CreateAll(gates []config.Gate) ([]gate.Gate, error)
}

// GateRunner abstracts parallel execution of gates.
type GateRunner interface {
	RunAll(ctx context.Context, gates []gate.Gate, failFast bool, gateNames []string) (*formatter.RunResult, error)
}
