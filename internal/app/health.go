package app

type HealthService struct{}

func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) Ping() string {
	return Version
}

func (s *HealthService) BuildInfo() BuildInfo {
	return Build()
}
