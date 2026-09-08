package service_info

import (
	"context"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"strings"
	"sync"
)

const (
	StatusAvailable   = "available"
	StatusUnavailable = "unavailable"
)

type ConnectedService struct{ Service, Version, Env, Status string }
type UseCase struct {
	workbench ports.ConnectedServiceInfoProvider
	mock      ports.MockGeneratorGateway
}

func NewUseCase(w ports.ConnectedServiceInfoProvider, m ports.MockGeneratorGateway) *UseCase {
	return &UseCase{workbench: w, mock: m}
}
func (u *UseCase) List(ctx context.Context) []ConnectedService {
	result := []ConnectedService{{Service: "service_ai_workbench", Status: StatusUnavailable}, {Service: "service_mock_generator", Status: StatusUnavailable}}
	providers := []ports.ConnectedServiceInfoProvider{u.workbench, u.mock}
	var wg sync.WaitGroup
	for i, p := range providers {
		if p == nil {
			continue
		}
		wg.Add(1)
		go func(i int, p ports.ConnectedServiceInfoProvider) {
			defer wg.Done()
			info, err := p.ServiceInfo(ctx)
			if err == nil && strings.TrimSpace(info.Version) != "" {
				result[i].Version = strings.TrimSpace(info.Version)
				result[i].Env = strings.TrimSpace(info.Env)
				result[i].Status = StatusAvailable
			}
		}(i, p)
	}
	wg.Wait()
	return result
}
