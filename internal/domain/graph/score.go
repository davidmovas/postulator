package graph

const (
	damping    = 0.85
	iterations = 30
)

func (g Graph) Score() map[string]float64 {
	n := len(g.ordered)
	scores := make(map[string]float64, n)
	if n == 0 {
		return scores
	}

	index := make(map[string]int, n)
	for i := range g.ordered {
		index[g.ordered[i].ID] = i
	}

	out := make([][]int, n)
	for i := range g.edges {
		e := &g.edges[i]
		if e.Status != StatusApproved {
			continue
		}
		from, to := index[e.FromEntityID], index[e.ToEntityID]
		out[from] = append(out[from], to)
		if e.Kind == EdgeRelated {
			out[to] = append(out[to], from)
		}
	}

	size := float64(n)
	rank := make([]float64, n)
	for i := range rank {
		rank[i] = 1 / size
	}

	for range iterations {
		next := make([]float64, n)
		dangling := 0.0
		for i, current := range rank {
			if len(out[i]) == 0 {
				dangling += current
				continue
			}
			share := current / float64(len(out[i]))
			for _, target := range out[i] {
				next[target] += share
			}
		}
		for i := range next {
			next[i] = (1-damping)/size + damping*(next[i]+dangling/size)
		}
		rank = next
	}

	highest := 0.0
	for _, value := range rank {
		highest = max(highest, value)
	}
	for i := range g.ordered {
		scores[g.ordered[i].ID] = rank[i] / highest
	}
	return scores
}
