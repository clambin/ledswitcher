package controller

type Health struct {
	Leader    bool
	Endpoints []string
}

func (c *Controller) Health() Health {
	return Health{
		Leader:    c.isLeader(),
		Endpoints: c.Broker.GetClients(),
	}
}
