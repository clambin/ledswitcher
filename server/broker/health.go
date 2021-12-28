package broker

type Health struct {
	Leader    bool
	Endpoints []string
	Current   string
}

func (lb *LEDBroker) Health() Health {
	return Health{
		Leader:    lb.IsLeading(),
		Endpoints: lb.GetClients(),
		Current:   lb.GetCurrentClient(),
	}
}
