package service_info

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

const (
	StatusAvailable   = "available"
	StatusUnavailable = "unavailable"
)

type ConnectedService struct {
	Service string
	Version string
	Env     string
	Status  string
}

type UseCase struct {
	workbench ports.ConnectedServiceInfoProvider
}

func NewUseCase(workbench ports.ConnectedServiceInfoProvider) *UseCase {
	return &UseCase{workbench: workbench}
}

func (u *UseCase) List(ctx context.Context) []ConnectedService {
	info, err := u.workbench.ServiceInfo(ctx)
	service := strings.TrimSpace(info.Service)
	if service == "" {
		service = "service_ai_workbench"
	}
	if err != nil || strings.TrimSpace(info.Version) == "" {
		return []ConnectedService{{Service: service, Status: StatusUnavailable}}
	}
	return []ConnectedService{{
		Service: service,
		Version: strings.TrimSpace(info.Version),
		Env:     strings.TrimSpace(info.Env),
		Status:  StatusAvailable,
	}}
}
